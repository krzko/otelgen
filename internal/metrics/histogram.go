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

// Histogram demonstrates how to record a distribution of individual values
func SimulateHistogram(mp *metric.MeterProvider, conf *Config, logger *zap.Logger) {
	c := *conf
	err := run(conf, logger, histogram(mp, c, logger))
	if err != nil {
		logger.Error("failed to run histogram", zap.Error(err))
	}
}

// histogram generates a histogram metric
func histogram(mp *metric.MeterProvider, c Config, logger *zap.Logger) WorkerFunc {
	return func(ctx context.Context) {
		name := fmt.Sprintf("%v.metrics.histogram", c.ServiceName)
		durRecorder, _ := mp.Meter(c.ServiceName).Int64Histogram(
			name,
			instrument.WithUnit("microseconds"),
			instrument.WithDescription("Histogram demonstrates how to record a distribution of individual values"),
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
				dur := time.Duration(rand.NormFloat64()*5000000) * time.Microsecond
				durRecorder.Record(ctx, dur.Microseconds())
				time.Sleep(time.Duration(float64(time.Second) / float64(c.Rate)))
			}
		} else {
			for {
				logger.Info("generating", zap.String("name", name))
				dur := time.Duration(rand.NormFloat64()*5000000) * time.Microsecond
				durRecorder.Record(ctx, dur.Microseconds())
				time.Sleep(time.Duration(float64(time.Second) / float64(c.Rate)))
			}
		}
	}
}
