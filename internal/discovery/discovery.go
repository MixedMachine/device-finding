package discovery

import (
	"context"
	"github.com/mixedmachine/device-finding/internal/devices"
	"log"
	"time"

	"github.com/grandcat/zeroconf"
)

// Register a new service (device)
func RegisterService(service, domain string, stop chan struct{}) {
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
func DiscoverServices(service, domain string, dm *devices.DeviceManager) {
	for {
		resolver, err := zeroconf.NewResolver(nil)
		if err != nil {
			log.Fatal("Failed to initialize resolver:", err)
		}

		entries := make(chan *zeroconf.ServiceEntry)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)

		go func(results <-chan *zeroconf.ServiceEntry, dm *devices.DeviceManager) {
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
