package metrics

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

type HistogramConfig struct {
	Name         string
	Description  string
	Unit         string
	Attributes   []attribute.KeyValue
	Temporality  metricdata.Temporality
	Bounds       []float64
	RecordMinMax bool
}

type HistogramDataPoint struct {
	ID            string
	Attributes    []attribute.KeyValue
	StartTimeUnix int64
	TimeUnix      int64
	Count         uint64
	Sum           float64
	Min           float64
	Max           float64
	BucketCounts  []uint64
	Exemplars     []Exemplar
}

func SimulateHistogram(mp metric.MeterProvider, config HistogramConfig, conf *Config, logger *zap.Logger) {
	c := *conf
	err := run(conf, logger, histogram(mp, config, c, logger))
	if err != nil {
		logger.Error("failed to run histogram", zap.Error(err))
	}
}

func histogram(mp metric.MeterProvider, config HistogramConfig, c Config, logger *zap.Logger) WorkerFunc {
	return func(ctx context.Context) {
		name := fmt.Sprintf("%v.metrics.histogram", c.ServiceName)
		logger.Debug("generating histogram", zap.String("name", name))

		histogram, err := mp.Meter(c.ServiceName).Float64Histogram(
			name,
			metric.WithUnit(config.Unit),
			metric.WithDescription(config.Description),
			metric.WithExplicitBucketBoundaries(config.Bounds...),
		)
		if err != nil {
			logger.Error("failed to create histogram", zap.Error(err))
			return
		}

		ticker := time.NewTicker(time.Duration(c.Rate) * time.Second)
		defer ticker.Stop()

		var cancel context.CancelFunc
		if c.TotalDuration > 0 {
			ctx, cancel = context.WithTimeout(ctx, c.TotalDuration)
			defer cancel()
		}

		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		startTime := time.Now()
		bucketCounts := make([]uint64, len(config.Bounds)+1)
		var count uint64
		var sum, min, max float64
		var exemplars []Exemplar

		for {
			select {
			case <-ctx.Done():
				logger.Info("Stopping histogram generation due to context cancellation")
				return
			case <-ticker.C:
				value := generateHistogramValue(r, config.Bounds)
				count++
				sum += value
				currentTime := time.Now()

				if config.RecordMinMax {
					if value < min || count == 1 {
						min = value
					}
					if value > max || count == 1 {
						max = value
					}
				}

				bucketIndex := findBucket(value, config.Bounds)
				bucketCounts[bucketIndex]++

				// Generate an exemplar
				exemplar := generateExemplar(r, value, currentTime)
				exemplars = append(exemplars, exemplar)

				// Limit the number of exemplars to keep memory usage in check
				if len(exemplars) > 10 {
					exemplars = exemplars[1:]
				}

				histogram.Record(ctx, value, metric.WithAttributes(config.Attributes...))

				// Log the current state of the histogram
				logger.Info("generating",
					zap.String("name", name),
					zap.Float64("value", value),
					zap.String("temporality", config.Temporality.String()),
					zap.Uint64("count", count),
					zap.Float64("sum", sum),
					zap.Float64("min", min),
					zap.Float64("max", max),
					zap.Int64("duration_seconds", currentTime.Sub(startTime).Milliseconds()/1000),
					zap.Reflect("bucket_counts", bucketCounts),
					zap.Int("exemplars_count", len(exemplars)),
				)

				dataPoint := HistogramDataPoint{
					ID:            uuid.New().String(),
					Attributes:    config.Attributes,
					StartTimeUnix: startTime.UnixNano(),
					TimeUnix:      currentTime.UnixNano(),
					Count:         count,
					Sum:           sum,
					Min:           min,
					Max:           max,
					BucketCounts:  bucketCounts,
					Exemplars:     exemplars,
				}

				if config.Temporality == metricdata.DeltaTemporality {
					// Reset for next delta
					startTime = currentTime
					count = 0
					sum = 0
					min = 0
					max = 0
					bucketCounts = make([]uint64, len(config.Bounds)+1)
					exemplars = nil
				}

				// Here you would send this dataPoint to your storage or processing system
				processHistogramDataPoint(dataPoint, logger)
			}
		}
	}
}

func generateHistogramValue(r *rand.Rand, bounds []float64) float64 {
	if len(bounds) == 0 {
		return r.Float64() * 100
	}
	maxBound := bounds[len(bounds)-1]
	// Generate values with a slight bias towards lower buckets
	return math.Pow(r.Float64(), 1.5) * maxBound * 1.1
}

func findBucket(value float64, bounds []float64) int {
	for i, bound := range bounds {
		if value <= bound {
			return i
		}
	}
	return len(bounds)
}

func processHistogramDataPoint(dataPoint HistogramDataPoint, logger *zap.Logger) {
	// This is where you would implement the logic to send the data point to your storage or processing system
	// For now, we'll just log some information about the data point
	logger.Info("Processing histogram data point",
		zap.String("id", dataPoint.ID),
		zap.Int64("start_time", dataPoint.StartTimeUnix),
		zap.Int64("time", dataPoint.TimeUnix),
		zap.Uint64("count", dataPoint.Count),
		zap.Float64("sum", dataPoint.Sum),
		zap.Float64("min", dataPoint.Min),
		zap.Float64("max", dataPoint.Max),
		zap.Int("bucket_count", len(dataPoint.BucketCounts)),
		zap.Int("exemplar_count", len(dataPoint.Exemplars)),
	)
}
