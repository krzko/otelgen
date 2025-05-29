// Package metrics provides types and functions for generating synthetic OpenTelemetry metrics.
package metrics

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

// SumConfig holds configuration for sum metric generation.
type SumConfig struct {
	Name        string
	Description string
	Unit        string
	Attributes  []attribute.KeyValue
	Temporality metricdata.Temporality
	IsMonotonic bool
}

// SimulateSum generates synthetic sum metrics using the provided configuration and logger.
func SimulateSum(mp metric.MeterProvider, sumConfig SumConfig, conf *Config, logger *zap.Logger) {
	if logger == nil {
		logger = zap.NewNop()
	}
	if conf == nil {
		logger.Error("config is nil, cannot run SimulateSum")
		return
	}
	if conf.Rate < 0 {
		logger.Error("rate must be non-negative for SimulateSum", zap.Int64("rate", int64(conf.Rate)))
		return
	}
	if conf.Rate == 0 {
		conf.Rate = 1 // Default to 1 if not set
	}
	c := *conf
	err := run(conf, logger, sum(mp, sumConfig, c, logger))
	if err != nil {
		logger.Error("failed to run sum", zap.Error(err))
	}
}

func sum(mp metric.MeterProvider, sc SumConfig, c Config, logger *zap.Logger) WorkerFunc {
	return func(ctx context.Context) {
		name := fmt.Sprintf("%v.metrics.sum", c.ServiceName)
		logger.Debug("generating sum", zap.String("name", name))
		counter, _ := mp.Meter(c.ServiceName).Int64Counter(
			name,
			metric.WithUnit(sc.Unit),
			metric.WithDescription(sc.Description),
		)

		r := NewRand()
		var exemplars []Exemplar
		var i int64

		if c.Rate == 0 {
			for {
				select {
				case <-ctx.Done():
					logger.Info("Stopping sum generation due to context cancellation")
					return
				default:
					i++
					value := i
					if !sc.IsMonotonic {
						value = (value % 100) - 50 // Oscillate between -50 and 49
					}
					exemplar := generateExemplar(r, float64(value), time.Now())
					exemplars = append(exemplars, exemplar)
					if len(exemplars) > 10 {
						exemplars = exemplars[1:]
					}
					logger.Info("generating",
						zap.String("name", name),
						zap.Int64("value", value),
						zap.String("temporality", sc.Temporality.String()),
						zap.Int("exemplars_count", len(exemplars)),
					)
					counter.Add(ctx, value, metric.WithAttributes(sc.Attributes...))
				}
			}
		} else {
			ticker := time.NewTicker(time.Second / time.Duration(c.Rate))
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					logger.Info("Stopping sum generation due to context cancellation")
					return
				case <-ticker.C:
					i++
					value := i
					if !sc.IsMonotonic {
						value = (value % 100) - 50 // Oscillate between -50 and 49
					}
					exemplar := generateExemplar(r, float64(value), time.Now())
					exemplars = append(exemplars, exemplar)
					if len(exemplars) > 10 {
						exemplars = exemplars[1:]
					}
					logger.Info("generating",
						zap.String("name", name),
						zap.Int64("value", value),
						zap.String("temporality", sc.Temporality.String()),
						zap.Int("exemplars_count", len(exemplars)),
					)
					counter.Add(ctx, value, metric.WithAttributes(sc.Attributes...))
				}
			}
		}
	}
}
