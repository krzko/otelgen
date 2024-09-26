package metrics

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

type GaugeConfig struct {
	Name        string
	Description string
	Unit        string
	Attributes  []attribute.KeyValue
	Min         float64
	Max         float64
	Temporality metricdata.Temporality
}

func SimulateGauge(mp metric.MeterProvider, gaugeConfig GaugeConfig, conf *Config, logger *zap.Logger) {
	c := *conf
	err := run(conf, logger, gauge(mp, gaugeConfig, c, logger))
	if err != nil {
		logger.Error("failed to run gauge", zap.Error(err))
	}
}

func gauge(mp metric.MeterProvider, gc GaugeConfig, c Config, logger *zap.Logger) WorkerFunc {
	return func(ctx context.Context) {
		name := fmt.Sprintf("%v.metrics.gauge", c.ServiceName)
		logger.Debug("generating gauge", zap.String("name", name))
		gauge, _ := mp.Meter(c.ServiceName).Float64ObservableGauge(
			name,
			metric.WithUnit(gc.Unit),
			metric.WithDescription(gc.Description),
		)

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
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

		ticker := time.NewTicker(time.Duration(c.Rate) * time.Second)
		defer ticker.Stop()

		var cancel context.CancelFunc
		if c.TotalDuration > 0 {
			ctx, cancel = context.WithTimeout(ctx, c.TotalDuration)
			defer cancel()
		}

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
				// The callback will be called automatically by the SDK
			}
		}
	}
}

func generateGaugeValue(min, max float64) float64 {
	amplitude := (max - min) / 2
	center := min + amplitude
	return center + amplitude*math.Sin(float64(time.Now().UnixNano())/1e9)
}
