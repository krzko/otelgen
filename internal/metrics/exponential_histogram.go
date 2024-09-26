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

type ExponentialHistogramConfig struct {
	Name          string
	Description   string
	Unit          string
	Attributes    []attribute.KeyValue
	Temporality   metricdata.Temporality
	Scale         int32
	MaxSize       float64
	RecordMinMax  bool
	ZeroThreshold float64
}

type ExponentialHistogramDataPoint struct {
	ID              string
	Attributes      []attribute.KeyValue
	StartTimeUnix   int64
	TimeUnix        int64
	Count           uint64
	Sum             float64
	Scale           int32
	ZeroCount       uint64
	PositiveBuckets map[int32]uint64
	NegativeBuckets map[int32]uint64
	Min             float64
	Max             float64
	Exemplars       []Exemplar
}

func SimulateExponentialHistogram(mp metric.MeterProvider, config ExponentialHistogramConfig, conf *Config, logger *zap.Logger) {
	c := *conf
	err := run(conf, logger, exponentialHistogram(mp, config, c, logger))
	if err != nil {
		logger.Error("failed to run exponential histogram", zap.Error(err))
	}
}

func exponentialHistogram(mp metric.MeterProvider, config ExponentialHistogramConfig, c Config, logger *zap.Logger) WorkerFunc {
	return func(ctx context.Context) {
		name := fmt.Sprintf("%v.metrics.exponential_histogram", c.ServiceName)
		logger.Debug("generating exponential histogram", zap.String("name", name))

		histogram, err := mp.Meter(c.ServiceName).Float64Histogram(
			name,
			metric.WithUnit(config.Unit),
			metric.WithDescription(config.Description),
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
		var min, max float64
		var zeroCount, totalCount uint64
		positiveBuckets := make(map[int32]uint64)
		negativeBuckets := make(map[int32]uint64)
		var sum float64
		var exemplars []Exemplar

		for {
			select {
			case <-ctx.Done():
				logger.Info("Stopping exponential histogram generation due to context cancellation")
				return
			case <-ticker.C:
				value := generateExponentialHistogramValue(r, config.MaxSize, config.ZeroThreshold)
				currentTime := time.Now()

				if config.RecordMinMax {
					if value < min || totalCount == 0 {
						min = value
					}
					if value > max || totalCount == 0 {
						max = value
					}
				}

				if math.Abs(value) <= config.ZeroThreshold {
					zeroCount++
				} else {
					index := mapToIndex(value, config.Scale)
					if value >= 0 {
						positiveBuckets[index]++
					} else {
						negativeBuckets[index]++
					}
				}
				totalCount++
				sum += value

				// Generate an exemplar
				exemplar := generateExemplar(r, value, currentTime)
				exemplars = append(exemplars, exemplar)

				// Limit the number of exemplars to keep memory usage in check
				if len(exemplars) > 10 {
					exemplars = exemplars[1:]
				}

				histogram.Record(ctx, value, metric.WithAttributes(config.Attributes...))
				logger.Info("generating",
					zap.String("name", name),
					zap.Float64("value", value),
					zap.String("temporality", config.Temporality.String()),
					zap.Int32("scale", config.Scale),
					zap.Uint64("zero_count", zeroCount),
					zap.Uint64("total_count", totalCount),
					zap.Float64("sum", sum),
					zap.Float64("min", min),
					zap.Float64("max", max),
					zap.Int("positive_buckets", len(positiveBuckets)),
					zap.Int("negative_buckets", len(negativeBuckets)),
					zap.Int("exemplars_count", len(exemplars)),
				)

				dataPoint := ExponentialHistogramDataPoint{
					ID:              uuid.New().String(),
					Attributes:      config.Attributes,
					StartTimeUnix:   startTime.UnixNano(),
					TimeUnix:        currentTime.UnixNano(),
					Count:           totalCount,
					Sum:             sum,
					Scale:           config.Scale,
					ZeroCount:       zeroCount,
					PositiveBuckets: positiveBuckets,
					NegativeBuckets: negativeBuckets,
					Min:             min,
					Max:             max,
					Exemplars:       exemplars,
				}

				if config.Temporality == metricdata.DeltaTemporality {
					// Reset for next delta
					startTime = currentTime
					totalCount = 0
					sum = 0
					min = 0
					max = 0
					zeroCount = 0
					positiveBuckets = make(map[int32]uint64)
					negativeBuckets = make(map[int32]uint64)
					exemplars = nil
				}

				// Here you would send this dataPoint to your storage or processing system
				processExponentialHistogramDataPoint(dataPoint, logger)
			}
		}
	}
}

func generateExponentialHistogramValue(r *rand.Rand, maxSize, zeroThreshold float64) float64 {
	// Generate a value using exponential distribution
	value := r.ExpFloat64() * maxSize / 10

	// Occasionally generate values near zero or maxSize
	if r.Float64() < 0.1 {
		if r.Float64() < 0.5 {
			value = r.Float64() * zeroThreshold
		} else {
			value = maxSize - r.Float64()*(maxSize/100)
		}
	}

	// Randomly make some values negative
	if r.Float64() < 0.3 {
		value = -value
	}

	return value
}

func mapToIndex(value float64, scale int32) int32 {
	if value == 0 {
		return 0
	}
	absValue := math.Abs(value)
	scaleFactor := math.Ldexp(math.Log2E, int(scale))
	return int32(math.Floor(math.Log(absValue) * scaleFactor))
}

func processExponentialHistogramDataPoint(dataPoint ExponentialHistogramDataPoint, logger *zap.Logger) {
	// This is where you would implement the logic to send the data point to your storage or processing system
	// For now, we'll just log some information about the data point
	logger.Info("Processing exponential histogram data point",
		zap.String("id", dataPoint.ID),
		zap.Int64("start_time", dataPoint.StartTimeUnix),
		zap.Int64("time", dataPoint.TimeUnix),
		zap.Uint64("count", dataPoint.Count),
		zap.Float64("sum", dataPoint.Sum),
		zap.Int32("scale", dataPoint.Scale),
		zap.Uint64("zero_count", dataPoint.ZeroCount),
		zap.Int("positive_buckets", len(dataPoint.PositiveBuckets)),
		zap.Int("negative_buckets", len(dataPoint.NegativeBuckets)),
		zap.Float64("min", dataPoint.Min),
		zap.Float64("max", dataPoint.Max),
		zap.Int("exemplar_count", len(dataPoint.Exemplars)),
	)
}
