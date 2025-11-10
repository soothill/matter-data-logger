package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/soothill/matter-data-logger/config"
	"github.com/soothill/matter-data-logger/discovery"
	"github.com/soothill/matter-data-logger/monitoring"
	"github.com/soothill/matter-data-logger/storage"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting Matter Power Data Logger")
	log.Printf("Discovery interval: %v", cfg.Matter.DiscoveryInterval)
	log.Printf("Poll interval: %v", cfg.Matter.PollInterval)

	// Initialize InfluxDB storage
	db, err := storage.NewInfluxDBStorage(
		cfg.InfluxDB.URL,
		cfg.InfluxDB.Token,
		cfg.InfluxDB.Organization,
		cfg.InfluxDB.Bucket,
	)
	if err != nil {
		log.Fatalf("Failed to initialize InfluxDB: %v", err)
	}
	defer db.Close()

	// Create device scanner
	scanner := discovery.NewScanner(cfg.Matter.ServiceType, cfg.Matter.Domain)

	// Create power monitor
	monitor := monitoring.NewPowerMonitor(cfg.Matter.PollInterval)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		log.Println("Initiating graceful shutdown...")
		cancel()
	}()

	// Start data writer goroutine
	go func() {
		for reading := range monitor.Readings() {
			if err := db.WriteReading(reading); err != nil {
				log.Printf("Failed to write reading to InfluxDB: %v", err)
			}
		}
	}()

	// Initial device discovery
	log.Println("Performing initial device discovery...")
	devices, err := scanner.Discover(ctx, 10*time.Second)
	if err != nil {
		log.Printf("Initial discovery failed: %v", err)
	} else {
		log.Printf("Discovered %d Matter devices", len(devices))
	}

	// Start monitoring power devices
	powerDevices := scanner.GetPowerDevices()
	log.Printf("Found %d devices with power measurement capability", len(powerDevices))

	if len(powerDevices) > 0 {
		monitor.Start(ctx, powerDevices)
	} else {
		log.Println("No power monitoring devices found. Will retry during periodic discovery.")
	}

	// Periodic device discovery
	discoveryTicker := time.NewTicker(cfg.Matter.DiscoveryInterval)
	defer discoveryTicker.Stop()

	// Main loop
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down...")
			db.Flush()
			return

		case <-discoveryTicker.C:
			log.Println("Performing periodic device discovery...")
			newDevices, err := scanner.Discover(ctx, 10*time.Second)
			if err != nil {
				log.Printf("Discovery failed: %v", err)
				continue
			}

			log.Printf("Discovery complete. Total devices: %d", len(scanner.GetDevices()))

			// Check for new power devices
			powerDevices := scanner.GetPowerDevices()
			if len(powerDevices) > 0 {
				log.Printf("Monitoring %d power devices", len(powerDevices))
				// Note: In production, you'd want to track which devices are already
				// being monitored to avoid starting duplicate monitoring goroutines
				for _, device := range newDevices {
					if device.HasPowerMeasurement() {
						log.Printf("Starting monitoring for new device: %s", device.Name)
						go monitor.Start(ctx, []*discovery.Device{device})
					}
				}
			}
		}
	}
}
