package main

import (
	"fmt"
	"os"
	"time"

	"github.com/grandcat/zeroconf"
)

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

	fmt.Println("Observing...")

	time.Sleep(2 * time.Second)

	go getDevicesMetrics(serviceName, deviceManager, stop)

	<-stop
}
