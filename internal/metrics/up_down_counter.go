package metrics

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// SimulateUpDownCounter demonstrates how to measure numbers that can go up and down
func SimulateUpDownCounter(mp metric.MeterProvider, conf *Config, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	if conf == nil {
		return errors.New("config is nil, cannot run SimulateUpDownCounter")
	}
	if err := conf.Validate(); err != nil {
		logger.Error("invalid config", zap.Error(err))
		return err
	}
	if conf.Rate < 0 {
		return fmt.Errorf("rate must be non-negative (got %f)", conf.Rate)
	}
	if conf.Rate == 0 {
		conf.Rate = 1 // Default to 1 if not set
	}
	c := *conf
	if err := run(conf, logger, upDownCounter(mp, c, logger)); err != nil {
		return fmt.Errorf("failed to run updowncounter: %w", err)
	}
	return nil
}

// upDownCounter generates a up down counter metric
func upDownCounter(mp metric.MeterProvider, c Config, logger *zap.Logger) WorkerFunc {
	return func(ctx context.Context) {
		name := fmt.Sprintf("%v.metrics.updowncounter", c.ServiceName)
		logger.Debug("generating updowncounter", zap.String("name", name))
		counter, _ := mp.Meter(c.ServiceName).Int64UpDownCounter(
			name,
		)

		r := NewRand()
		var exemplars []Exemplar
		var i int64

		if c.Rate == 0 {
			for {
				select {
				case <-ctx.Done():
					logger.Info("Stopping updowncounter generation due to context cancellation")
					return
				default:
					i++
					value := (i % 200) - 100 // Oscillate between -100 and 99
					exemplar := generateExemplar(r, float64(value), time.Now())
					exemplars = append(exemplars, exemplar)
					if len(exemplars) > 10 {
						exemplars = exemplars[1:]
					}
					logger.Info("generating",
						zap.String("name", name),
						zap.Int64("value", value),
						zap.Int("exemplars_count", len(exemplars)),
					)
					counter.Add(ctx, value)
				}
			}
		} else {
			ticker := time.NewTicker(time.Second / time.Duration(c.Rate))
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					logger.Info("Stopping updowncounter generation due to context cancellation")
					return
				case <-ticker.C:
					i++
					value := (i % 200) - 100 // Oscillate between -100 and 99
					exemplar := generateExemplar(r, float64(value), time.Now())
					exemplars = append(exemplars, exemplar)
					if len(exemplars) > 10 {
						exemplars = exemplars[1:]
					}
					logger.Info("generating",
						zap.String("name", name),
						zap.Int64("value", value),
						zap.Int("exemplars_count", len(exemplars)),
					)
					counter.Add(ctx, value)
				}
			}
		}
	}
}
