package main

import (
	"fmt"
	"sync"

	"github.com/grandcat/zeroconf"
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

