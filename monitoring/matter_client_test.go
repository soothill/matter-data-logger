// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package monitoring_test

import (
	"testing"

	"github.com/soothill/matter-data-logger/discovery"
	"github.com/soothill/matter-data-logger/monitoring"
)

func BenchmarkReadPower(b *testing.B) {
	device := &discovery.Device{
		Name: "Test Device",
	}
	client := monitoring.NewMatterClient(device)

	for i := 0; i < b.N; i++ {
		_, err := client.ReadPower()
		if err != nil {
			b.Fatal(err)
		}
	}
}
