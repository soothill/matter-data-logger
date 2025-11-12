# Extending the Matter Data Logger

This document provides examples of how to extend the functionality of the Matter Data Logger.

## Adding a New Storage Backend

To add a new storage backend, you need to implement the `Storage` interface defined in `pkg/interfaces/storage.go`.

```go
// pkg/interfaces/storage.go
type Storage interface {
    Write(ctx context.Context, data Point) error
    Close() error
}
```

**Example: Implementing a Prometheus Remote Write Backend**

1.  **Create a new package**: `storage/prometheus`
2.  **Implement the `Storage` interface**:

    ```go
    // storage/prometheus/prometheus.go
    package prometheus

    import (
        // ...
    )

    type PrometheusStorage struct {
        // ...
    }

    func NewPrometheusStorage(...) (*PrometheusStorage, error) {
        // ...
    }

    func (s *PrometheusStorage) Write(ctx context.Context, data interfaces.Point) error {
        // Convert the data point to a Prometheus remote write sample and send it.
    }

    func (s *PrometheusStorage) Close() error {
        // Clean up any resources.
    }
    ```

3.  **Integrate the new backend**: Modify `main.go` to allow selecting the new storage backend via configuration.

## Supporting Additional Matter Clusters

To support monitoring additional Matter clusters:

1.  **Identify the Cluster ID**: Find the ID of the cluster you want to support in the Matter specification.
2.  **Update the Discovery Logic**: Modify the `discovery` package to identify devices that support the new cluster.
3.  **Implement the Monitoring Logic**: In the `monitoring` package, add code to read the attributes of the new cluster.

## Adding Custom Metrics

You can easily add new Prometheus metrics by using the `Metrics` interface in `pkg/interfaces/metrics.go`.

**Example: Adding a Metric for Cache Size**

1.  **Define the new metric**: In `pkg/metrics/metrics.go`, add a new Prometheus metric.

    ```go
    // pkg/metrics/metrics.go
    s.cacheSize = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "matter_data_logger_cache_size_bytes",
        Help: "The current size of the local cache in bytes.",
    })
    ```

2.  **Update the metric**: In the `storage` package, update the metric whenever the cache size changes.

    ```go
    // storage/cache.go
    func (c *Cache) Add(item cacheItem) {
        // ...
        c.metrics.RecordCacheSize(c.currentSize)
    }
