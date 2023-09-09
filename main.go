package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

type DeviceManager struct {
	activeDevices map[string]*zeroconf.ServiceEntry
	mutex         sync.Mutex
}

func (dm *DeviceManager) AddDevice(device *zeroconf.ServiceEntry) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	_, exists := dm.activeDevices[device.Instance]
	if !exists {
		fmt.Printf("Device joined: %s\n", device.Instance)
	}

	dm.activeDevices[device.Instance] = device
}

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
		"_http._tcp",                // service type
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

func main() {
	domain := "local."
	serviceName, _ := os.Hostname()

	fmt.Println("My name is", serviceName)

	deviceManager := &DeviceManager{
		activeDevices: make(map[string]*zeroconf.ServiceEntry),
	}

	stop := make(chan struct{})

	go registerService(serviceName, domain, stop)

	time.Sleep(2 * time.Second)

	go discoverServices("_http._tcp", domain, deviceManager)

	fmt.Println("Zeroconf initialized.Observing...")
	<-stop
}
