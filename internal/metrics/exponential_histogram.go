package metrics

import (
	"context"
	"errors"
	"fmt"
	"math"
	randv2 "math/rand/v2"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

// ExponentialHistogramConfig holds configuration for exponential histogram metrics.
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

// ExponentialHistogramDataPoint represents a single data point for an exponential histogram metric.
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

// SimulateExponentialHistogram generates synthetic exponential histogram metrics using the provided configuration and logger.
func SimulateExponentialHistogram(mp metric.MeterProvider, config ExponentialHistogramConfig, conf *Config, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	if conf == nil {
		return errors.New("config is nil, cannot run SimulateExponentialHistogram")
	}
	if err := conf.Validate(); err != nil {
		logger.Error("invalid config", zap.Error(err))
		return err
	}
	c := *conf
	if err := run(conf, logger, exponentialHistogram(mp, config, c, logger)); err != nil {
		return fmt.Errorf("failed to run exponential histogram: %w", err)
	}
	return nil
}

func exponentialHistogram(mp metric.MeterProvider, config ExponentialHistogramConfig, c Config, logger *zap.Logger) WorkerFunc {
	return func(ctx context.Context) {
		if err := c.Validate(); err != nil {
			logger.Error("invalid config", zap.Error(err))
			return
		}
		if c.Rate <= 0 {
			logger.Error("rate must be positive")
			return
		}
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

		r := NewRand()

		startTime := time.Now()
		var minVal, maxVal float64
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
					if value < minVal || totalCount == 0 {
						minVal = value
					}
					if value > maxVal || totalCount == 0 {
						maxVal = value
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
					zap.Float64("min", minVal),
					zap.Float64("max", maxVal),
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
					Min:             minVal,
					Max:             maxVal,
					Exemplars:       exemplars,
				}

				minVal = value
				maxVal = value

				// Reset min and max appropriately for delta temporality:
				if config.Temporality == metricdata.DeltaTemporality {
					startTime = currentTime
					totalCount = 0
					sum = 0
					minVal = math.MaxFloat64  // Set to max possible float value for correct min calculation in next round
					maxVal = -math.MaxFloat64 // Set to min possible value for correct max calculation in next round
					zeroCount = 0
					positiveBuckets = make(map[int32]uint64)
					negativeBuckets = make(map[int32]uint64)
					exemplars = nil
				}

				processExponentialHistogramDataPoint(dataPoint, logger)
			}
		}
	}
}

func generateExponentialHistogramValue(r *randv2.Rand, maxSize, zeroThreshold float64) float64 {
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
	// Calculate the base: 2^(2^(-scale))
	base := math.Exp2(math.Exp2(-float64(scale)))
	return int32(math.Floor(math.Log(absValue) / math.Log(base)))
}

func processExponentialHistogramDataPoint(dataPoint ExponentialHistogramDataPoint, logger *zap.Logger) {
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
