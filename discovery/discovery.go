// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package discovery

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/soothill/matter-data-logger/pkg/logger"
)

// Device represents a discovered Matter device
type Device struct {
	Name      string
	Address   net.IP
	Port      int
	TXTRecord map[string]string
	Hostname  string
}

// HasPowerMeasurement checks if the device supports power measurement
func (d *Device) HasPowerMeasurement() bool {
	// Check TXT records for power measurement capability
	// Matter devices advertise their clusters in TXT records
	if clusters, ok := d.TXTRecord["C"]; ok {
		// Cluster 0x0B04 is the Electrical Measurement cluster
		// Cluster 0x0091 is the Electrical Power Measurement cluster (new)
		return strings.Contains(clusters, "0B04") ||
		       strings.Contains(clusters, "B04") ||
		       strings.Contains(clusters, "0091") ||
		       strings.Contains(clusters, "91")
	}
	return false
}

// GetDeviceID returns a unique identifier for the device
func (d *Device) GetDeviceID() string {
	if d.TXTRecord != nil {
		if id, ok := d.TXTRecord["D"]; ok && id != "" {
			return id
		}
	}
	return fmt.Sprintf("%s:%d", d.Address.String(), d.Port)
}

// Scanner handles Matter device discovery via mDNS
type Scanner struct {
	serviceType string
	domain      string
	devices     map[string]*Device
}

// NewScanner creates a new device scanner
func NewScanner(serviceType, domain string) *Scanner {
	return &Scanner{
		serviceType: serviceType,
		domain:      domain,
		devices:     make(map[string]*Device),
	}
}

// Discover performs a single discovery scan for Matter devices
func (s *Scanner) Discover(ctx context.Context, timeout time.Duration) ([]*Device, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	discoveredDevices := make([]*Device, 0)

	go func() {
		for entry := range entries {
			device := s.parseServiceEntry(entry)
			if device != nil {
				deviceID := device.GetDeviceID()
				s.devices[deviceID] = device
				discoveredDevices = append(discoveredDevices, device)
				logger.Info().
					Str("device_id", deviceID).
					Str("device_name", device.Name).
					Str("address", device.Address.String()).
					Int("port", device.Port).
					Bool("has_power_measurement", device.HasPowerMeasurement()).
					Msg("Discovered Matter device")
			}
		}
	}()

	discoverCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = resolver.Browse(discoverCtx, s.serviceType, s.domain, entries)
	if err != nil {
		return nil, fmt.Errorf("failed to browse: %w", err)
	}

	<-discoverCtx.Done()

	return discoveredDevices, nil
}

// parseServiceEntry converts a zeroconf service entry to a Device
func (s *Scanner) parseServiceEntry(entry *zeroconf.ServiceEntry) *Device {
	// Validate entry
	if entry == nil {
		return nil
	}

	if len(entry.AddrIPv4) == 0 && len(entry.AddrIPv6) == 0 {
		return nil
	}

	// Prefer IPv4, fallback to IPv6
	var addr net.IP
	if len(entry.AddrIPv4) > 0 {
		addr = entry.AddrIPv4[0]
	} else {
		addr = entry.AddrIPv6[0]
	}

	txtRecord := make(map[string]string)
	for _, txt := range entry.Text {
		parts := strings.SplitN(txt, "=", 2)
		if len(parts) == 2 {
			txtRecord[parts[0]] = parts[1]
		}
	}

	return &Device{
		Name:      entry.Instance,
		Address:   addr,
		Port:      entry.Port,
		TXTRecord: txtRecord,
		Hostname:  entry.HostName,
	}
}

// GetDevices returns all discovered devices
func (s *Scanner) GetDevices() []*Device {
	devices := make([]*Device, 0, len(s.devices))
	for _, device := range s.devices {
		devices = append(devices, device)
	}
	return devices
}

// GetPowerDevices returns only devices that support power measurement
func (s *Scanner) GetPowerDevices() []*Device {
	powerDevices := make([]*Device, 0)
	for _, device := range s.devices {
		if device.HasPowerMeasurement() {
			powerDevices = append(powerDevices, device)
		}
	}
	return powerDevices
}
