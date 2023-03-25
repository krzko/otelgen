package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type WorkerFunc func(ctx context.Context)

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

// Run runs the worker
func (w *Worker) Run(ctx context.Context, workerFunc WorkerFunc) error {
	if w.totalDuration == 0 {
		// w.numMetrics = 0
		w.totalDuration = 86400 * time.Second // 24 hours
	} else if w.numMetrics == 0 {
		w.logger.Error("either `metrics` or `duration` must be greater than 0")
		return fmt.Errorf("either `metrics` or `duration` must be greater than 0")
	}

	running := atomic.NewBool(true)
	errChan := make(chan error, 1)
	for i := 0; i < 1; i++ {
		w.wg.Add(1)

		go func() {
			defer w.wg.Done()
			workerFunc(ctx)
		}()
	}

	if w.totalDuration > 0 {
		w.logger.Info("generation duration", zap.Float64("seconds", w.totalDuration.Seconds()))
		w.logger.Info("generation rate", zap.Float64("per second", float64(w.limitPerSecond)))
		time.Sleep(w.totalDuration)
		running.Store(false)
	}
	w.wg.Wait()

	// Check if there's an error in the error channel
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}
