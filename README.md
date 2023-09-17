# Device Finder

*Mixed Machine*
<br />
*<mixedmachine.dev@gmail.com>*

## Description

This repository contains code for a simple Device Finder and Manager using Zeroconf for service discovery and registration. Additionally, it opens up communication channel between devices for resource share and possible additional protocol messages in the future. It is written in Go and leverages the grandcat/zeroconf library and udp socket communication.

## Features

- Registers the current device as a service.
- Discovers other services (devices) on the same network.
- Keeps track of active devices.
- Logs when a device joins or leaves the network.
- Opens up a communication channel between devices.
- Sends a message to all devices on the network.
- Sends device metrics including CPU, Memory, and Network usage.
