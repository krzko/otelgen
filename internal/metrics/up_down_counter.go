package metrics

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

// SimulateUpDownCounter demonstrates how to measure numbers that can go up and down
func SimulateUpDownCounter(ctx context.Context, mp *metric.MeterProvider, conf *Config, logger *zap.Logger) {
	c := *conf
	err := run(conf, logger, upDownCounter(mp, c, logger))
	if err != nil {
		logger.Error("failed to run up-down-counter", zap.Error(err))
	}
}

// upDownCounter generates a up down counter metric
func upDownCounter(mp *metric.MeterProvider, c Config, logger *zap.Logger) WorkerFunc {
	return func(ctx context.Context) {
		name := fmt.Sprintf("%v.metrics.up_down_counter", c.ServiceName)
		counter, _ := mp.Meter(c.ServiceName).Int64UpDownCounter(
			name,
			instrument.WithUnit("1"),
			instrument.WithDescription("UpDownCounter demonstrates how to measure numbers that can go up and down"),
		)

		if c.TotalDuration > 0 {
			logger.Info("generation duration", zap.Float64("seconds", c.TotalDuration.Seconds()))

		loop:
			for timeout := time.After(c.TotalDuration); ; {
				select {
				case <-timeout:
					break loop
				default:
				}
				logger.Info("generating", zap.String("name", name))
				if rand.Float64() >= 0.5 {
					counter.Add(ctx, +1)
				} else {
					counter.Add(ctx, -1)
				}
				time.Sleep(time.Duration(c.Rate) * time.Second)
			}
		} else {
			for {
				logger.Info("generating", zap.String("name", name))
				if rand.Float64() >= 0.5 {
					counter.Add(ctx, +1)
				} else {
					counter.Add(ctx, -1)
				}
				time.Sleep(time.Duration(c.Rate) * time.Second)
			}
		}
	}
}
