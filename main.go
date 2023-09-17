package main

import (
	"github.com/mixedmachine/device-finding/internal/communication"
	"github.com/mixedmachine/device-finding/internal/devices"
	"github.com/mixedmachine/device-finding/internal/discovery"

	"fmt"
	"os"
	"time"
)

func main() {
	domain := "local."
	serviceName, _ := os.Hostname()

	fmt.Println("My name is", serviceName)

	deviceManager := devices.NewDeviceManager()

	stop := make(chan struct{})

	go discovery.RegisterService(serviceName, domain, stop)

	go communication.ListenForDevices(serviceName)

	time.Sleep(2 * time.Second)

	go discovery.DiscoverServices("_myudp._udp", domain, deviceManager)

	fmt.Println("Observing...")

	time.Sleep(2 * time.Second)

	go communication.GetDevicesMetrics(serviceName, deviceManager, stop)

	<-stop
}
