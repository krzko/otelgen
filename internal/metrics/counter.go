package metrics

import (
	"context"
	"fmt"
	"time"

	apiMetric "go.opentelemetry.io/otel/metric"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

// Counter demonstrates how to measure non-decreasing int64s
func SimulateCounter(mp *sdkMetric.MeterProvider, conf *Config, logger *zap.Logger) {
	c := *conf
	err := run(conf, logger, counter(mp, c, logger))
	if err != nil {
		logger.Error("failed to run counter", zap.Error(err))
	}
}

// counter generates a counter metric
func counter(mp *sdkMetric.MeterProvider, c Config, logger *zap.Logger) WorkerFunc {
	return func(ctx context.Context) {
		name := fmt.Sprintf("%v.metrics.counter", c.ServiceName)
		logger.Debug("generating counter", zap.String("name", name))
		counter, _ := mp.Meter(c.ServiceName).Int64Counter(
			name,
			apiMetric.WithUnit("1"),
			apiMetric.WithDescription("Counter demonstrates how to measure non-decreasing numbers"),
		)

		var i int64
		if c.TotalDuration > 0 {
		loop:
			for timeout := time.After(c.TotalDuration); ; {
				select {
				case <-timeout:
					break loop
				default:
				}
				i++
				logger.Info("generating", zap.String("name", name))
				counter.Add(ctx, i)
				time.Sleep(time.Duration(c.Rate) * time.Second)
			}
		} else {
			for {
				i++
				logger.Info("generating", zap.String("name", name))
				counter.Add(ctx, i)
				time.Sleep(time.Duration(c.Rate) * time.Second)
			}
		}
	}
}
