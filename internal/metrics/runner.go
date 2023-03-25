package metrics

import (
	"context"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

// run is a function that runs a worker
func run(c *Config, logger *zap.Logger, workerFunc WorkerFunc, mp *metric.MeterProvider) error {
	w := NewWorker(c, logger)
	w.Run(context.Background(), workerFunc)
	return nil
}
