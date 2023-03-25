package metrics

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// run is a function that runs a worker
func run(c *Config, logger *zap.Logger, workerFunc WorkerFunc) error {
	w := NewWorker(c, logger)
	if err := w.Run(context.Background(), workerFunc); err != nil {
		return fmt.Errorf("failed to run worker: %w", err)
	}
	return nil
}
