// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package discovery_test

import (
	"testing"

	"github.com/soothill/matter-data-logger/discovery"
)

func FuzzGetDeviceID(f *testing.F) {
	f.Add("device-name", "instance-name")
	f.Fuzz(func(t *testing.T, name, instance string) {
		device := &discovery.Device{
			Name:         name,
			InstanceName: instance,
		}
		deviceID := device.GetDeviceID()
		if deviceID == "" {
			t.Errorf("GetDeviceID returned an empty string for name %q and instance %q", name, instance)
		}
	})
}
