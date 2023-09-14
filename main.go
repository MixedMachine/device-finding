package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	snet "github.com/shirou/gopsutil/net"
)

// DeviceManager keeps track of active devices
type DeviceManager struct {
	activeDevices map[string]*zeroconf.ServiceEntry
	mutex         sync.Mutex
}

// Add a new device to the list of active devices
func (dm *DeviceManager) AddDevice(device *zeroconf.ServiceEntry) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	_, exists := dm.activeDevices[device.Instance]
	if !exists {
		fmt.Printf("Device joined: %s\n", device.Instance)
	}

	dm.activeDevices[device.Instance] = device
}

// Remove inactive devices from the list of active devices
func (dm *DeviceManager) RemoveInactiveDevice(newEntries map[string]*zeroconf.ServiceEntry) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	for instance := range dm.activeDevices {
		if _, exists := newEntries[instance]; !exists {
			delete(dm.activeDevices, instance)
			fmt.Printf("Device left: %s\n", instance)
		}
	}
}

// Register a new service (device)
func registerService(service, domain string, stop chan struct{}) {
	server, err := zeroconf.Register(
		service,                     // service instance name
		"_myudp._udp",               // service type
		domain,                      // service domain
		8080,                        // service port
		[]string{"txtv=0", "lo=la"}, // service metadata
		nil,                         // register on all network interfaces
	)
	if err != nil {
		log.Fatal("Failed to register service:", err)
	}
	defer server.Shutdown()

	<-stop
}

// Discover services (devices)
func discoverServices(service, domain string, dm *DeviceManager) {
	for {
		resolver, err := zeroconf.NewResolver(nil)
		if err != nil {
			log.Fatal("Failed to initialize resolver:", err)
		}

		entries := make(chan *zeroconf.ServiceEntry)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)

		go func(results <-chan *zeroconf.ServiceEntry, dm *DeviceManager) {
			newEntries := make(map[string]*zeroconf.ServiceEntry)
			for entry := range results {
				newEntries[entry.Instance] = entry
				dm.AddDevice(entry)
			}

			dm.RemoveInactiveDevice(newEntries)
		}(entries, dm)

		err = resolver.Browse(ctx, service, domain, entries)
		if err != nil {
			log.Fatal("Failed to browse:", err)
		}

		<-ctx.Done()
		cancel() // Release resources
		time.Sleep(10 * time.Second)
	}
}

// Get device metrics
func getDeviceMetrics() string {
	var metrics []string

	// Get CPU Usage (over a 1-second interval)
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return ""
	}

	// For simplicity, average the percentages if there are multiple CPUs
	var totalPercent float64
	for _, percent := range percentages {
		totalPercent += percent
	}
	avgPercent := totalPercent / float64(len(percentages))
	metrics = append(metrics, strconv.FormatFloat(avgPercent, 'f', 2, 64))

	// Get Memory Info
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return ""
	}
	metrics = append(metrics, strconv.FormatUint(memInfo.Available, 10), strconv.FormatUint(memInfo.Total, 10))

	// Get Network Info
	netInfo, err := snet.IOCounters(false)
	if err != nil {
		return ""
	}

	for _, netStat := range netInfo {
		metrics = append(metrics, strconv.FormatUint(netStat.BytesSent, 10), strconv.FormatUint(netStat.BytesRecv, 10))
	}

	// Construct CSV-like string
	csvMetrics := strings.Join(metrics, ",")

	return csvMetrics
}

// Handle received message
func handleReceivedMessage(self, msg string) {
	protocol := strings.Split(msg, " ")
	if len(protocol) != 4 {
		fmt.Println("Invalid protocol")
		return
	}

	device := protocol[0]
	deviceIP := protocol[1]
	msgType := protocol[2]
	data := protocol[3]

	fmt.Printf("Received message from %s: %s %s\n", device, msgType, data)
	if msgType == "REQ" {
		switch data {
		case "metrics":
			metrics := getDeviceMetrics()
			conn, err := net.Dial("udp", fmt.Sprintf("%s:4256", deviceIP))
			defer conn.Close()
			if err != nil {
				fmt.Println("Failed to dial UDP:", err)
				return
			}

			_, err = conn.Write(buildMessage(self, getIPv4Address(), "RES", metrics))
			if err != nil {
				fmt.Println("Failed to write to UDP:", err)
			}
		default:
			fmt.Println("Unknown request")
		}
	}
}

// Listen for devices
func listenForDevices(self string) {
	addr := net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: 4256,
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP: %v", err)
	}

	defer conn.Close()

	for {
		buf := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatalf("Failed to read from UDP: %v", err)
		}
		handleReceivedMessage(self, string(buf[:n]))
	}
}

func buildMessage(self, selfIP, msgType, msg string) []byte {
	return []byte(fmt.Sprintf("%s %s %s %s", self, selfIP, msgType, msg))
}

// Get devices metrics
func getDevicesMetrics(self string, dm *DeviceManager, stop chan struct{}) {
	for {
		dm.mutex.Lock()
		wg := &sync.WaitGroup{}
		for _, device := range dm.activeDevices {
			wg.Add(1)
			go func(device *zeroconf.ServiceEntry) {
				defer wg.Done()
				if device.Instance == self {
					return
				}
				if len(device.AddrIPv4) == 0 {
					fmt.Println("No IPv4 address found for device", device.Instance)
					return
				}
				url := fmt.Sprintf("%s:4256", device.AddrIPv4[0])

				conn, err := net.Dial("udp", url)
				if err != nil {
					fmt.Println("Failed to dial UDP:", err)
					return
				}

				_, err = conn.Write(buildMessage(self, getIPv4Address(), "REQ", "metrics"))
				if err != nil {
					fmt.Println("Failed to write to UDP:", err)
				}
				conn.Close()
			}(device)
		}
		dm.mutex.Unlock()
		wg.Wait() // Wait for all goroutines to finish
		time.Sleep(10 * time.Second)
	}
}

func getIPv4Address() string {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, address := range addresses {
		ipnet, ok := address.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
			continue
		}

		return ipnet.IP.String()
	}

	return ""
}

func main() {
	domain := "local."
	serviceName, _ := os.Hostname()

	fmt.Println("My name is", serviceName)

	deviceManager := &DeviceManager{
		activeDevices: make(map[string]*zeroconf.ServiceEntry),
	}

	stop := make(chan struct{})

	go registerService(serviceName, domain, stop)

	go listenForDevices(serviceName)

	time.Sleep(2 * time.Second)

	go discoverServices("_myudp._udp", domain, deviceManager)

	fmt.Println("Zeroconf initialized. Observing...")

	time.Sleep(2 * time.Second)

	go getDevicesMetrics(serviceName, deviceManager, stop)

	<-stop
}
