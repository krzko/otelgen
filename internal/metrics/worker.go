package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// WorkerFunc defines the signature for a worker function that processes metrics.
type WorkerFunc func(ctx context.Context)

// Worker represents a worker that generates metrics.
type Worker struct {
	numMetrics     int             // how many metrics the worker has to generate (only when duration==0)
	totalDuration  time.Duration   // how long to run the test for (overrides `numMetrics`)
	limitPerSecond rate.Limit      // how many metrics per second to generate
	wg             *sync.WaitGroup // notify when done
	logger         *zap.Logger
}

// NewWorker creates a new worker
func NewWorker(c *Config, logger *zap.Logger) *Worker {
	return &Worker{
		numMetrics:     c.NumMetrics,
		totalDuration:  c.TotalDuration,
		limitPerSecond: rate.Limit(c.Rate),
		wg:             &sync.WaitGroup{},
		logger:         logger,
	}
}

// run is a function that runs a worker
func run(c *Config, logger *zap.Logger, workerFunc WorkerFunc) error {
	w := NewWorker(c, logger)
	if err := w.Run(context.Background(), workerFunc); err != nil {
		return fmt.Errorf("failed to run worker: %w", err)
	}
	return nil
}

// Run runs the worker
func (w *Worker) Run(ctx context.Context, workerFunc WorkerFunc) error {
	// If no duration is set, default to 24 hours
	if w.totalDuration == 0 {
		w.totalDuration = 86400 * time.Second // 24 hours
	}

	// Wrap the context with a timeout for duration-based cancellation
	var cancel context.CancelFunc
	if w.totalDuration > 0 {
		ctx, cancel = context.WithTimeout(ctx, w.totalDuration)
		defer cancel()
	}

	errChan := make(chan error, 1)
	for i := 0; i < 1; i++ {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			workerFunc(ctx)
		}()
	}

	w.logger.Info("generation duration", zap.Float64("seconds", w.totalDuration.Seconds()))
	w.logger.Info("generation rate", zap.Float64("per second", float64(w.limitPerSecond)))

	// Wait for all workers to finish (they should exit when ctx is done)
	w.wg.Wait()

	// Check if there's an error in the error channel
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}
