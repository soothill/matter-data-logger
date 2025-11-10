// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package monitoring

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/soothill/matter-data-logger/discovery"
	"github.com/soothill/matter-data-logger/pkg/logger"
)

// PowerReading represents a power consumption measurement
type PowerReading struct {
	DeviceID    string
	DeviceName  string
	Timestamp   time.Time
	Power       float64 // Power in watts
	Voltage     float64 // Voltage in volts
	Current     float64 // Current in amperes
	Energy      float64 // Cumulative energy in kWh
}

// PowerMonitor handles power consumption monitoring
type PowerMonitor struct {
	pollInterval      time.Duration
	readings          chan *PowerReading
	monitoredDevices  map[string]context.CancelFunc
	deviceMutex       sync.RWMutex
}

// NewPowerMonitor creates a new power monitor
func NewPowerMonitor(pollInterval time.Duration) *PowerMonitor {
	return &PowerMonitor{
		pollInterval:     pollInterval,
		readings:         make(chan *PowerReading, 100),
		monitoredDevices: make(map[string]context.CancelFunc),
	}
}

// Start begins monitoring the given devices
func (pm *PowerMonitor) Start(ctx context.Context, devices []*discovery.Device) {
	logger.Info().Msgf("Starting power monitoring for %d devices", len(devices))

	for _, device := range devices {
		pm.StartMonitoringDevice(ctx, device)
	}
}

// StartMonitoringDevice starts monitoring a single device if not already monitored
func (pm *PowerMonitor) StartMonitoringDevice(ctx context.Context, device *discovery.Device) bool {
	deviceID := device.GetDeviceID()

	pm.deviceMutex.Lock()
	defer pm.deviceMutex.Unlock()

	// Check if device is already being monitored
	if _, exists := pm.monitoredDevices[deviceID]; exists {
		logger.Debug().Str("device_id", deviceID).Str("device_name", device.Name).
			Msg("Device already being monitored, skipping")
		return false
	}

	// Create a cancellable context for this device
	deviceCtx, cancel := context.WithCancel(ctx)
	pm.monitoredDevices[deviceID] = cancel

	logger.Info().Str("device_id", deviceID).Str("device_name", device.Name).
		Msg("Starting monitoring for new device")

	go pm.monitorDevice(deviceCtx, device)
	return true
}

// StopMonitoringDevice stops monitoring a specific device
func (pm *PowerMonitor) StopMonitoringDevice(deviceID string) {
	pm.deviceMutex.Lock()
	defer pm.deviceMutex.Unlock()

	if cancel, exists := pm.monitoredDevices[deviceID]; exists {
		cancel()
		delete(pm.monitoredDevices, deviceID)
		logger.Info().Str("device_id", deviceID).Msg("Stopped monitoring device")
	}
}

// IsMonitoring checks if a device is currently being monitored
func (pm *PowerMonitor) IsMonitoring(deviceID string) bool {
	pm.deviceMutex.RLock()
	defer pm.deviceMutex.RUnlock()
	_, exists := pm.monitoredDevices[deviceID]
	return exists
}

// GetMonitoredDeviceCount returns the number of devices being monitored
func (pm *PowerMonitor) GetMonitoredDeviceCount() int {
	pm.deviceMutex.RLock()
	defer pm.deviceMutex.RUnlock()
	return len(pm.monitoredDevices)
}

// monitorDevice continuously polls a single device for power data
func (pm *PowerMonitor) monitorDevice(ctx context.Context, device *discovery.Device) {
	ticker := time.NewTicker(pm.pollInterval)
	defer ticker.Stop()

	deviceID := device.GetDeviceID()
	logger.Info().Str("device_id", deviceID).Str("device_name", device.Name).
		Msg("Monitoring device")

	// Clean up when done
	defer func() {
		pm.deviceMutex.Lock()
		delete(pm.monitoredDevices, deviceID)
		pm.deviceMutex.Unlock()
		logger.Info().Str("device_id", deviceID).Msg("Stopped monitoring device")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reading, err := pm.readPower(device)
			if err != nil {
				logger.Error().Err(err).Str("device_id", deviceID).Str("device_name", device.Name).
					Msg("Error reading power from device")
				continue
			}

			select {
			case pm.readings <- reading:
			default:
				logger.Warn().Str("device_id", deviceID).Str("device_name", device.Name).
					Msg("Readings channel full, dropping reading")
			}
		}
	}
}

// readPower reads power consumption from a Matter device
// NOTE: This is a simplified implementation. In a real system, you would:
// 1. Establish a Matter session with the device
// 2. Read attributes from the Electrical Measurement cluster (0x0B04)
// 3. Parse the Matter TLV-encoded response
// For demonstration purposes, this generates simulated data
func (pm *PowerMonitor) readPower(device *discovery.Device) (*PowerReading, error) {
	// In a production system, you would implement actual Matter protocol communication here
	// This would involve:
	// - PASE/CASE session establishment
	// - Reading cluster attributes via Matter protocol
	// - Handling Matter message encoding/decoding

	// For now, we'll simulate realistic power readings
	// You would replace this with actual Matter cluster reads

	// Simulate reading from Matter Electrical Measurement cluster
	// Typical attributes:
	// - ActivePower (0x050B): signed 16-bit, in watts
	// - RMSVoltage (0x0505): unsigned 16-bit, in volts
	// - RMSCurrent (0x0508): unsigned 16-bit, in milliamps

	baseLoad := 10.0 + rand.Float64()*90.0  // 10-100W base load
	variation := (rand.Float64() - 0.5) * 10.0 // ±5W variation
	power := baseLoad + variation

	voltage := 120.0 + (rand.Float64()-0.5)*2.0 // 119-121V
	current := power / voltage

	reading := &PowerReading{
		DeviceID:   device.GetDeviceID(),
		DeviceName: device.Name,
		Timestamp:  time.Now(),
		Power:      power,
		Voltage:    voltage,
		Current:    current,
		Energy:     0, // Would need to track cumulative energy
	}

	logger.Debug().
		Str("device_id", reading.DeviceID).
		Str("device_name", reading.DeviceName).
		Float64("power_w", reading.Power).
		Float64("voltage_v", reading.Voltage).
		Float64("current_a", reading.Current).
		Msg("Power reading")

	return reading, nil
}

// Readings returns the channel for receiving power readings
func (pm *PowerMonitor) Readings() <-chan *PowerReading {
	return pm.readings
}

// TODO: Implement actual Matter protocol communication
// This would require:
// 1. Matter CHIP library integration (chip-tool or matter.js bindings)
// 2. Device commissioning and session management
// 3. Cluster attribute reading via Matter Interaction Model
// 4. TLV encoding/decoding for Matter messages
//
// Example Matter cluster attributes for power measurement:
// - Electrical Measurement Cluster (0x0B04):
//   - MeasurementType (0x0000)
//   - ActivePower (0x050B): Power in watts
//   - RMSVoltage (0x0505): Voltage in volts
//   - RMSCurrent (0x0508): Current in milliamps
//   - ApparentPower (0x050F)
//   - PowerFactor (0x0510)
//
// - Electrical Power Measurement Cluster (0x0091) [newer]:
//   - PowerMode (0x0000)
//   - NumberOfMeasurementTypes (0x0001)
//   - Accuracy (0x0002)
//   - Ranges (0x0003)
//   - Voltage (0x0004)
//   - ActiveCurrent (0x0005)
//   - ActivePower (0x0008)
//   - Energy (0x000B)
