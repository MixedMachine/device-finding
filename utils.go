package main

import (
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	snet "github.com/shirou/gopsutil/net"
)

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


