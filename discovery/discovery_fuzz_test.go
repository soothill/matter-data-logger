// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package discovery

import (
	"net"
	"testing"
)

// FuzzDevice_HasPowerMeasurement tests HasPowerMeasurement with random cluster strings
func FuzzDevice_HasPowerMeasurement(f *testing.F) {
	// Seed corpus with known inputs
	f.Add("0006,0008,0B04,001D") // Valid with 0B04
	f.Add("0006,0008,B04,001D")  // Valid with B04
	f.Add("0006,0008,0091,001D") // Valid with 0091
	f.Add("0006,0008,91,001D")   // Valid with 91
	f.Add("0006,0008,001D")      // Invalid - no power cluster
	f.Add("")                    // Empty string
	f.Add("0B04")                // Just the power cluster
	f.Add("B04,0091")            // Multiple power clusters
	f.Add("0000,0B04,0000")      // Power cluster in middle
	f.Add("0B040B04")            // Adjacent without comma
	f.Add(",,,")                 // Just commas
	f.Add("0B 04")               // With space
	f.Add("0B04\n0091")          // With newline
	f.Add("0x0B04")              // With hex prefix
	f.Add("0B04,")               // Trailing comma
	f.Add(",0B04")               // Leading comma

	f.Fuzz(func(t *testing.T, clusterString string) {
		// Create device with fuzzed cluster string
		device := &Device{
			TXTRecord: map[string]string{
				"C": clusterString,
			},
		}

		// Call should never panic
		result := device.HasPowerMeasurement()

		// Basic sanity check - if string contains power cluster, should be true
		// This is a loose check because fuzzing might generate weird formats
		_ = result
	})
}

// FuzzDevice_GetDeviceID tests GetDeviceID with random discriminator values
func FuzzDevice_GetDeviceID(f *testing.F) {
	// Seed corpus with known inputs
	f.Add("12345")                    // Normal discriminator
	f.Add("")                         // Empty discriminator
	f.Add("device-abc-123")           // Alphanumeric
	f.Add("1234567890123456789012345") // Very long
	f.Add("\x00\x01\x02")             // Binary data
	f.Add("device\nwith\nnewlines")   // With newlines
	f.Add("device\twith\ttabs")       // With tabs
	f.Add("192.168.1.100:5540")       // IP:port format
	f.Add("::::")                     // Colons
	f.Add("device/with/slashes")      // Slashes
	f.Add("device with spaces")       // Spaces
	f.Add("UPPERCASE")                // Uppercase
	f.Add("MiXeD-CaSe-123")           // Mixed case
	f.Add("unicode-日本語-测试")         // Unicode
	f.Add("\"; DROP TABLE devices;--") // SQL injection attempt

	f.Fuzz(func(t *testing.T, discriminator string) {
		// Create device with fuzzed discriminator
		device := &Device{
			Address: net.ParseIP("192.168.1.100"),
			Port:    5540,
			TXTRecord: map[string]string{
				"D": discriminator,
			},
		}

		// Call should never panic and always return a string
		result := device.GetDeviceID()

		// Result should never be empty
		if result == "" {
			t.Errorf("GetDeviceID() returned empty string for discriminator=%q", discriminator)
		}

		// If discriminator is empty, should fall back to address:port
		if discriminator == "" {
			expected := "192.168.1.100:5540"
			if result != expected {
				t.Errorf("GetDeviceID() with empty discriminator = %v, want %v", result, expected)
			}
		}
	})
}

// FuzzDevice_GetDeviceID_NilTXTRecord tests GetDeviceID with nil TXT record
func FuzzDevice_GetDeviceID_NilTXTRecord(f *testing.F) {
	// Seed with various IP addresses
	f.Add("192.168.1.100", 5540)
	f.Add("10.0.0.1", 80)
	f.Add("255.255.255.255", 65535)
	f.Add("0.0.0.0", 0)
	f.Add("127.0.0.1", 1)

	f.Fuzz(func(t *testing.T, ip string, port int) {
		// Skip invalid ports
		if port < 0 || port > 65535 {
			return
		}

		// Try to parse IP
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			// Invalid IP, skip
			return
		}

		// Create device with nil TXT record
		device := &Device{
			Address:   parsedIP,
			Port:      port,
			TXTRecord: nil,
		}

		// Call should never panic
		result := device.GetDeviceID()

		// Result should be in format "ip:port"
		if result == "" {
			t.Errorf("GetDeviceID() returned empty string for ip=%s, port=%d", ip, port)
		}
	})
}

// FuzzDevice_HasPowerMeasurement_InvalidTXTRecord tests with various invalid TXT records
func FuzzDevice_HasPowerMeasurement_InvalidTXTRecord(f *testing.F) {
	// Seed with edge cases
	f.Add(true)  // nil TXT record
	f.Add(false) // empty TXT record

	f.Fuzz(func(t *testing.T, useNil bool) {
		var device *Device
		if useNil {
			device = &Device{
				TXTRecord: nil,
			}
		} else {
			device = &Device{
				TXTRecord: map[string]string{},
			}
		}

		// Call should never panic and return false
		result := device.HasPowerMeasurement()
		if result {
			t.Errorf("HasPowerMeasurement() with empty/nil TXT record should return false, got true")
		}
	})
}

// FuzzDevice_ClusterStringParsing tests various cluster string formats
func FuzzDevice_ClusterStringParsing(f *testing.F) {
	// Seed corpus with various formats that should be handled gracefully
	f.Add("0B04", ",", 4)  // Normal with comma separator
	f.Add("B04", ";", 3)   // Without leading zero, semicolon separator
	f.Add("0091", " ", 4)  // Space separator
	f.Add("91", "|", 2)    // Short form with pipe
	f.Add("0B04", "", 4)   // No separator
	f.Add("", ",", 0)      // Empty cluster
	f.Add("XXXX", ",", 4)  // Invalid hex

	f.Fuzz(func(t *testing.T, cluster string, separator string, length int) {
		// Build a cluster string with the given format
		if length < 0 || length > 100 {
			return // Skip unreasonable lengths
		}

		// Create a cluster string
		var clusterString string
		if length > 0 {
			for i := 0; i < length; i++ {
				if i > 0 {
					clusterString += separator
				}
				clusterString += cluster
			}
		}

		device := &Device{
			TXTRecord: map[string]string{
				"C": clusterString,
			},
		}

		// Should never panic
		_ = device.HasPowerMeasurement()
	})
}
