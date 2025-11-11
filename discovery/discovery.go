// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package discovery provides Matter device discovery via mDNS (multicast DNS).
//
// This package implements automatic discovery of Matter devices on the local network
// using mDNS service discovery. It identifies devices with power measurement capabilities
// by examining their Matter cluster information.
//
// # Matter Protocol
//
// Matter devices advertise themselves using mDNS with the service type "_matter._tcp".
// Each device includes TXT records containing:
//   - D: Discriminator (device identifier)
//   - VP: Vendor ID and Product ID
//   - CM: Commissioning Mode
//   - CL: Cluster information (indicates supported features)
//
// # Power Measurement Detection
//
// The scanner automatically identifies devices with power measurement by checking
// for these Matter clusters in the device's TXT records:
//   - 0x0B04: Electrical Measurement Cluster (older standard)
//   - 0x0091: Electrical Power Measurement Cluster (newer standard)
//
// # Thread Safety
//
// All scanner operations are thread-safe and use read-write locks to protect
// the internal device map. Multiple goroutines can safely call scanner methods
// concurrently.
//
// # Example Usage
//
//	scanner := discovery.NewScanner("_matter._tcp", "local.")
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	devices, err := scanner.Discover(ctx, 10*time.Second)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get only devices with power measurement capability
//	powerDevices := scanner.GetPowerDevices()
//	for _, device := range powerDevices {
//	    fmt.Printf("Found power device: %s\n", device.Name)
//	}
package discovery

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
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
	mu          sync.RWMutex // Protects devices map
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
//
// # Concurrency Architecture
//
// This function uses a producer-consumer pattern with goroutines and channels:
//
// 1. Producer: zeroconf.Resolver.Browse() (external library)
//    - Scans network for mDNS/DNS-SD advertisements
//    - Sends ServiceEntry records to the 'entries' channel
//    - Runs until context timeout expires
//
// 2. Consumer: Local goroutine (lines 128-154)
//    - Receives ServiceEntry records from the 'entries' channel
//    - Parses each entry into a Device struct
//    - Updates two data structures concurrently
//
// # Synchronization Strategy
//
// Two separate mutexes protect two different data structures:
//
// 1. s.mu (RWMutex): Protects s.devices map (Scanner-wide state)
//    - This map persists across multiple Discover() calls
//    - Allows safe concurrent reads (GetDevices, GetPowerDevices)
//    - Write-locked only during device insertion
//
// 2. mu (Mutex): Protects discoveredDevices slice (function-local state)
//    - This slice only tracks newly discovered devices in this scan
//    - Returned to caller after scan completes
//    - Simpler mutex since no concurrent reads needed
//
// # Data Flow
//
//	┌──────────────┐     entries      ┌─────────────────┐
//	│   zeroconf   │─────channel──────>│  Consumer      │
//	│   Resolver   │   (buffered 10)   │  Goroutine     │
//	└──────────────┘                   └─────────────────┘
//	                                            │
//	                                            v
//	                                    ┌───────────────┐
//	                                    │ Parse Entry   │
//	                                    └───────────────┘
//	                                            │
//	                                            v
//	                                   ┌────────────────┐
//	                                   │ Update Maps    │
//	                                   │ (mutex-locked) │
//	                                   └────────────────┘
//
// # Buffering Rationale
//
// The 'entries' channel is buffered (capacity 10) to prevent blocking the
// zeroconf resolver when device parsing is slower than discovery rate.
// This prevents packet loss during bursts of mDNS advertisements.
func (s *Scanner) Discover(ctx context.Context, timeout time.Duration) ([]*Device, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver: %w", err)
	}

	// Buffered channel to prevent blocking zeroconf resolver
	entries := make(chan *zeroconf.ServiceEntry, 10)
	discoveredDevices := make([]*Device, 0)
	var mu sync.Mutex // Protects discoveredDevices slice (function-local)
	var wg sync.WaitGroup

	// Consumer goroutine: processes discovered devices concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Loop until entries channel is closed by zeroconf resolver
		for entry := range entries {
			device := s.parseServiceEntry(entry)
			if device != nil {
				deviceID := device.GetDeviceID()

				// Critical Section 1: Update Scanner-wide devices map
				// Uses RWMutex to allow concurrent reads from other goroutines
				s.mu.Lock()
				s.devices[deviceID] = device
				s.mu.Unlock()

				// Critical Section 2: Update function-local discoveredDevices slice
				// Uses simple Mutex since no concurrent readers
				mu.Lock()
				discoveredDevices = append(discoveredDevices, device)
				mu.Unlock()

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
	wg.Wait() // Wait for goroutine to finish processing all entries

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
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]*Device, 0, len(s.devices))
	for _, device := range s.devices {
		devices = append(devices, device)
	}
	return devices
}

// GetPowerDevices returns only devices that support power measurement
func (s *Scanner) GetPowerDevices() []*Device {
	s.mu.RLock()
	defer s.mu.RUnlock()

	powerDevices := make([]*Device, 0)
	for _, device := range s.devices {
		if device.HasPowerMeasurement() {
			powerDevices = append(powerDevices, device)
		}
	}
	return powerDevices
}

// GetDeviceByID returns a device by its ID, or nil if not found
func (s *Scanner) GetDeviceByID(deviceID string) *Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.devices[deviceID]
}
