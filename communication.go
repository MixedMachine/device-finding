package main

import (
	"fmt"
	"net"
	"time"
	"strings"
	"sync"

	"github.com/grandcat/zeroconf"
)

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
		panic("Failed to listen on UDP: " + err.Error())
	}

	defer conn.Close()

	for {
		buf := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			panic("Failed to read from UDP: " + err.Error())
		}
		handleReceivedMessage(self, string(buf[:n]))
	}
}

// build message based on protocol
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

