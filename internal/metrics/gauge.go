// Package metrics provides types and functions for generating synthetic OpenTelemetry metrics.
package metrics

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

// GaugeConfig holds configuration for gauge metric generation.
type GaugeConfig struct {
	Name        string
	Description string
	Unit        string
	Attributes  []attribute.KeyValue
	Min         float64
	Max         float64
	Temporality metricdata.Temporality
}

// SimulateGauge generates synthetic gauge metrics using the provided configuration and logger.
func SimulateGauge(mp metric.MeterProvider, gaugeConfig GaugeConfig, conf *Config, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	if conf == nil {
		return errors.New("config is nil, cannot run SimulateGauge")
	}
	if err := conf.Validate(); err != nil {
		logger.Error("invalid config", zap.Error(err))
		return err
	}
	if conf.Rate < 0 {
		return fmt.Errorf("rate must be non-negative (got %f)", conf.Rate)
	}
	c := *conf
	if err := run(conf, logger, gauge(mp, gaugeConfig, c, logger)); err != nil {
		return fmt.Errorf("failed to run gauge: %w", err)
	}
	return nil
}

func gauge(mp metric.MeterProvider, gc GaugeConfig, c Config, logger *zap.Logger) WorkerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(ctx context.Context) {
		if err := c.Validate(); err != nil {
			logger.Error("invalid config", zap.Error(err))
			return
		}
		if c.Rate < 0 {
			logger.Error("rate must be non-negative")
			return
		}
		name := fmt.Sprintf("%v.metrics.gauge", c.ServiceName)
		logger.Debug("generating gauge", zap.String("name", name))
		gauge, _ := mp.Meter(c.ServiceName).Float64ObservableGauge(
			name,
			metric.WithUnit(gc.Unit),
			metric.WithDescription(gc.Description),
		)

		r := NewRand()
		var exemplars []Exemplar

		_, err := mp.Meter(c.ServiceName).RegisterCallback(func(_ context.Context, o metric.Observer) error {
			value := generateGaugeValue(gc.Min, gc.Max)
			o.ObserveFloat64(gauge, value, metric.WithAttributes(gc.Attributes...))
			return nil
		}, gauge)
		if err != nil {
			logger.Error("failed to register callback", zap.Error(err))
			return
		}

		if c.Rate == 0 {
			for {
				select {
				case <-ctx.Done():
					logger.Info("Stopping gauge generation due to context cancellation")
					return
				default:
					value := generateGaugeValue(gc.Min, gc.Max)
					exemplar := generateExemplar(r, value, time.Now())
					exemplars = append(exemplars, exemplar)
					if len(exemplars) > 10 {
						exemplars = exemplars[1:]
					}
					logger.Info("generating",
						zap.String("name", name),
						zap.Float64("value", value),
						zap.String("temporality", gc.Temporality.String()),
						zap.Int("exemplars_count", len(exemplars)),
					)
				}
			}
		} else {
			ticker := time.NewTicker(time.Second / time.Duration(c.Rate))
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					logger.Info("Stopping gauge generation due to context cancellation")
					return
				case <-ticker.C:
					value := generateGaugeValue(gc.Min, gc.Max)
					exemplar := generateExemplar(r, value, time.Now())
					exemplars = append(exemplars, exemplar)
					if len(exemplars) > 10 {
						exemplars = exemplars[1:]
					}
					logger.Info("generating",
						zap.String("name", name),
						zap.Float64("value", value),
						zap.String("temporality", gc.Temporality.String()),
						zap.Int("exemplars_count", len(exemplars)),
					)
				}
			}
		}
	}
}

// generateGaugeValue returns a random float64 value between min and max.
func generateGaugeValue(minVal, maxVal float64) float64 {
	amplitude := (maxVal - minVal) / 2
	center := minVal + amplitude
	return center + amplitude*math.Sin(float64(time.Now().UnixNano())/1e9)
}
