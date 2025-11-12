// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package app_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/soothill/matter-data-logger/app"
	"github.com/soothill/matter-data-logger/config"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type AppIntegrationTestSuite struct {
	suite.Suite
	influxDBContainer testcontainers.Container
	influxDBURL       string
}

func TestAppIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AppIntegrationTestSuite))
}

func (s *AppIntegrationTestSuite) SetupSuite() {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "influxdb:2.7",
		ExposedPorts: []string{"8086/tcp"},
		Env: map[string]string{
			"DOCKER_INFLUXDB_INIT_MODE":      "setup",
			"DOCKER_INFLUXDB_INIT_USERNAME":  "testuser",
			"DOCKER_INFLUXDB_INIT_PASSWORD":  "testpassword",
			"DOCKER_INFLUXDB_INIT_ORG":       "testorg",
			"DOCKER_INFLUXDB_INIT_BUCKET":    "testbucket",
			"DOCKER_INFLUXDB_INIT_ADMIN_TOKEN": "testtoken",
		},
		WaitingFor: wait.ForHTTP("/ping").WithPort("8086"),
	}
	influxDBContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	s.Require().NoError(err)
	s.influxDBContainer = influxDBContainer

	ip, err := influxDBContainer.Host(ctx)
	s.Require().NoError(err)
	port, err := influxDBContainer.MappedPort(ctx, "8086")
	s.Require().NoError(err)
	s.influxDBURL = "http://" + ip + ":" + port.Port()
}

func (s *AppIntegrationTestSuite) TearDownSuite() {
	if s.influxDBContainer != nil {
		s.Require().NoError(s.influxDBContainer.Terminate(context.Background()))
	}
}

func (s *AppIntegrationTestSuite) TestAppLifecycle() {
	// Create a temporary config file
	configFile, err := os.CreateTemp("", "config-*.yaml")
	s.Require().NoError(err)
	defer os.Remove(configFile.Name())

	configContent := `
influxdb:
  url: %s
  token: testtoken
  organization: testorg
  bucket: testbucket
matter:
  discovery_interval: 1s
  poll_interval: 1s
`
	_, err = configFile.WriteString(fmt.Sprintf(configContent, s.influxDBURL))
	s.Require().NoError(err)
	configFile.Close()

	cfg, err := config.Load(configFile.Name())
	s.Require().NoError(err)

	app, err := app.New(cfg, "9091", configFile.Name())
	s.Require().NoError(err)

	done := make(chan struct{})
	go func() {
		app.Run()
		close(done)
	}()

	// Wait for the app to start
	time.Sleep(2 * time.Second)

	// Send shutdown signal
	p, err := os.FindProcess(os.Getpid())
	s.Require().NoError(err)
	s.Require().NoError(p.Signal(os.Interrupt))

	// Wait for the app to shut down
	select {
	case <-done:
		// App shut down gracefully
	case <-time.After(5 * time.Second):
		s.T().Fatal("App did not shut down gracefully")
	}
}
