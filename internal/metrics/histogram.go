package metrics

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	randv2 "math/rand/v2"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

// HistogramConfig holds configuration for histogram metrics.
type HistogramConfig struct {
	Name         string
	Description  string
	Unit         string
	Attributes   []attribute.KeyValue
	Temporality  metricdata.Temporality
	Bounds       []float64
	RecordMinMax bool
}

// HistogramDataPoint represents a single data point for a histogram metric.
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

// SimulateHistogram generates synthetic histogram metrics using the provided configuration and logger.
func SimulateHistogram(mp metric.MeterProvider, config HistogramConfig, conf *Config, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	if conf == nil {
		return errors.New("config is nil, cannot run SimulateHistogram")
	}
	if err := conf.Validate(); err != nil {
		logger.Error("invalid config", zap.Error(err))
		return err
	}
	c := *conf
	if err := run(conf, logger, histogram(mp, config, c, logger)); err != nil {
		return fmt.Errorf("failed to run histogram: %w", err)
	}
	return nil
}

func histogram(mp metric.MeterProvider, config HistogramConfig, c Config, logger *zap.Logger) WorkerFunc {
	return func(ctx context.Context) {
		if err := c.Validate(); err != nil {
			logger.Error("invalid config", zap.Error(err))
			return
		}
		if c.Rate < 0 {
			logger.Error("rate must be non-negative")
			return
		}
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

		var ticker *time.Ticker
		if c.Rate > 0 {
			ticker = time.NewTicker(time.Second / time.Duration(c.Rate))
			defer ticker.Stop()
		}

		var cancel context.CancelFunc
		if c.TotalDuration > 0 {
			ctx, cancel = context.WithTimeout(ctx, c.TotalDuration)
			defer cancel()
		}

		startTime := time.Now()
		bucketCounts := make([]uint64, len(config.Bounds)+1)
		var count uint64
		var sum, minVal, maxVal float64
		var exemplars []Exemplar

		for {
			select {
			case <-ctx.Done():
				logger.Info("Stopping histogram generation due to context cancellation")
				return
			default:
				if c.Rate == 0 || (ticker != nil && selectTicker(ticker)) {
					value := generateHistogramValue(NewRand(), config.Bounds)
					count++
					sum += value
					currentTime := time.Now()

					if config.RecordMinMax {
						if value < minVal || count == 1 {
							minVal = value
						}
						if value > maxVal || count == 1 {
							maxVal = value
						}
					}

					bucketIndex := findBucket(value, config.Bounds)
					bucketCounts[bucketIndex]++

					// Generate an exemplar
					exemplar := generateExemplar(NewRand(), value, currentTime)
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
						zap.Float64("min", minVal),
						zap.Float64("max", maxVal),
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
						Min:           minVal,
						Max:           maxVal,
						BucketCounts:  bucketCounts,
						Exemplars:     exemplars,
					}

					if config.Temporality == metricdata.DeltaTemporality {
						// Reset for next delta
						startTime = currentTime
						count = 0
						sum = 0
						minVal = 0
						maxVal = 0
						bucketCounts = make([]uint64, len(config.Bounds)+1)
						exemplars = nil
					}

					processHistogramDataPoint(dataPoint, logger)
				} else if ticker != nil {
					// Wait for ticker.C
					<-ticker.C
				}
			}
		}
	}
}

// Helper to check if ticker.C is ready
func selectTicker(ticker *time.Ticker) bool {
	select {
	case <-ticker.C:
		return true
	default:
		return false
	}
}

func generateHistogramValue(r *randv2.Rand, bounds []float64) float64 {
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
