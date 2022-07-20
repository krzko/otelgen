package metrics

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type Config struct {
	NumMetrics    int
	Rate          int64
	TotalDuration time.Duration
	ServiceName   string

	// OTLP config
	Endpoint string
	Insecure bool
	UseHTTP  bool
	Headers  HeaderValue
}

type HeaderValue map[string]string

var _ flag.Value = (*HeaderValue)(nil)

func (v *HeaderValue) String() string {
	return ""
}

func (v *HeaderValue) Set(s string) error {
	kv := strings.SplitN(s, "=", 2)
	if len(kv) != 2 {
		return fmt.Errorf("value should be of the format key=value")
	}
	(*v)[kv[0]] = kv[1]
	return nil
}

// Counter demonstrates how to measure non-decreasing numbers, for example,
// number of requests or connections.
func Counter(ctx context.Context, m metric.Meter, c *Config, logger *zap.Logger) {
	name := fmt.Sprintf("%v.metrics.counter", c.ServiceName)
	logger.Debug("Generating counter", zap.String("name", name))
	counter, _ := m.SyncInt64().Counter(
		name,
		instrument.WithUnit("1"),
		instrument.WithDescription("Counter demonstrates how to measure non-decreasing numbers"),
	)

	var i int64 = 0
	for {
		i++
		logger.Info("Generating", zap.String("name", name))
		counter.Add(ctx, i)
		time.Sleep(30 * time.Second)
	}
}

// UpDownCounter demonstrates how to measure numbers that can go up and down, for example,
// number of goroutines or customers.
func UpDownCounter(ctx context.Context, m metric.Meter, c *Config, logger *zap.Logger) {
	name := fmt.Sprintf("%v.metrics.up_down_counter", c.ServiceName)
	counter, _ := m.SyncInt64().UpDownCounter(
		name,
		instrument.WithUnit("1"),
		instrument.WithDescription("UpDownCounter demonstrates how to measure numbers that can go up and down"),
	)

	for {
		logger.Info("Generating", zap.String("name", name))
		if rand.Float64() >= 0.5 {
			counter.Add(ctx, +1)
		} else {
			counter.Add(ctx, -1)
		}

		time.Sleep(60 * time.Second)
	}
}

// Histogram demonstrates how to record a distribution of individual values, for example,
// request or query timings. With this instrument you get total number of records,
// avg/min/max values, and heatmaps/percentiles.
func Histogram(ctx context.Context, m metric.Meter, c *Config, logger *zap.Logger) {
	name := fmt.Sprintf("%v.metrics.histogram", c.ServiceName)
	durRecorder, _ := m.SyncInt64().Histogram(
		name,
		instrument.WithUnit("microseconds"),
		instrument.WithDescription("Histogram demonstrates how to record a distribution of individual values"),
	)

	for {
		logger.Info("Generating", zap.String("name", name))
		dur := time.Duration(rand.NormFloat64()*5000000) * time.Microsecond
		durRecorder.Record(ctx, dur.Microseconds())

		time.Sleep(time.Second)
	}
}

// CounterObserver demonstrates how to measure monotonic (non-decreasing) numbers,
// for example, number of requests or connections.
func CounterObserver(ctx context.Context, m metric.Meter, c *Config, logger *zap.Logger) {
	name := fmt.Sprintf("%v.metrics.counter_observer", c.ServiceName)
	counter, _ := m.AsyncInt64().Counter(
		name,
		instrument.WithUnit("1"),
		instrument.WithDescription("CounterObserver demonstrates how to measure monotonic (non-decreasing) numbers"),
	)

	logger.Info("Generating", zap.String("name", name))
	var number int64
	if err := m.RegisterCallback(
		[]instrument.Asynchronous{
			counter,
		},
		// SDK periodically calls this function to collect data.
		func(ctx context.Context) {
			number++
			counter.Observe(ctx, number)
		},
	); err != nil {
		panic(err)
	}
}

// UpDownCounterObserver demonstrates how to measure numbers that can go up and down,
// for example, number of goroutines or customers.
func UpDownCounterObserver(ctx context.Context, m metric.Meter, c *Config, logger *zap.Logger) {
	name := fmt.Sprintf("%v.metrics.up_down_counter_async", c.ServiceName)
	counter, err := m.AsyncInt64().UpDownCounter(
		name,
		instrument.WithUnit("1"),
		instrument.WithDescription("UpDownCounterObserver demonstrates how to measure numbers that can go up and down"),
	)
	if err != nil {
		panic(err)
	}

	logger.Info("Generating", zap.String("name", name))
	if err := m.RegisterCallback(
		[]instrument.Asynchronous{
			counter,
		},
		func(ctx context.Context) {
			num := runtime.NumGoroutine()
			counter.Observe(ctx, int64(num))
		},
	); err != nil {
		panic(err)
	}
}

// GaugeObserver demonstrates how to measure non-additive numbers that can go up and down,
// for example, cache hit rate or memory utilization.
func GaugeObserver(ctx context.Context, m metric.Meter, c *Config, logger *zap.Logger) {
	name := fmt.Sprintf("%v.metrics.gauge_observer", c.ServiceName)
	gauge, _ := m.AsyncFloat64().Gauge(
		name,
		instrument.WithUnit("1"),
		instrument.WithDescription("GaugeObserver demonstrates how to measure non-additive numbers that can go up and down"),
	)

	logger.Info("Generating", zap.String("name", name))
	if err := m.RegisterCallback(
		[]instrument.Asynchronous{
			gauge,
		},
		func(ctx context.Context) {
			gauge.Observe(ctx, rand.Float64())
		},
	); err != nil {
		panic(err)
	}
}

// CounterWithLabels demonstrates how to add different labels ("hits" and "misses")
// to measurements. Using this simple trick, you can get number of hits, misses,
// sum = hits + misses, and hit_rate = hits / (hits + misses).
func CounterWithLabels(ctx context.Context, m metric.Meter, c *Config, logger *zap.Logger) {
	name := fmt.Sprintf("%v.metrics.cache", c.ServiceName)
	counter, _ := m.SyncInt64().Counter(
		name,
		instrument.WithDescription("CounterWithLabels demonstrates how to add different labels"),
	)
	for {
		logger.Info("Generating", zap.String("name", name))
		if rand.Float64() < 0.3 {
			// increment hits
			counter.Add(ctx, 1, attribute.String("type", "hits"))
		} else {
			// increments misses
			counter.Add(ctx, 1, attribute.String("type", "misses"))
		}

		time.Sleep(time.Millisecond)
	}
}

// CounterObserverAdvanced demonstrates how to measure monotonic (non-decreasing) numbers,
// for example, number of requests or connections.
func CounterObserverAdvanced(ctx context.Context, m metric.Meter, c *Config, logger *zap.Logger) {
	name := fmt.Sprintf("%v.metrics.cache_hits", c.ServiceName)
	// stats is our data source updated by some library.
	var stats struct {
		Hits   int64 // atomic
		Misses int64 // atomic
	}

	hitsCounter, _ := m.AsyncInt64().Counter(
		name,
		instrument.WithDescription("CounterObserverAdvanced cache hit"))
	missesCounter, _ := m.AsyncInt64().Counter("some.prefix.cache_misses",
		instrument.WithDescription("CounterObserverAdvanced cache miss"))

	if err := m.RegisterCallback(
		[]instrument.Asynchronous{
			hitsCounter,
			missesCounter,
		},
		// SDK periodically calls this function to collect data.
		func(ctx context.Context) {
			hitsCounter.Observe(ctx, atomic.LoadInt64(&stats.Hits))
			missesCounter.Observe(ctx, atomic.LoadInt64(&stats.Misses))
		},
	); err != nil {
		panic(err)
	}

	for {
		logger.Info("Generating", zap.String("name", name))
		if rand.Float64() < 0.3 {
			atomic.AddInt64(&stats.Misses, 1)
		} else {
			atomic.AddInt64(&stats.Hits, 1)
		}

		time.Sleep(time.Millisecond)
	}
}

// Run executes the test scenario.
func Run(ctx context.Context, exp *otlpmetric.Exporter, m metric.Meter, c *Config, logger *zap.Logger) func() {
	limit := rate.Limit(c.Rate)
	if c.Rate == 0 {
		limit = rate.Inf
		logger.Info("generation of metrics isn't being throttled")
	} else {
		logger.Info("generation of metrics is limited", zap.Float64("per-second", float64(limit)))
	}

	pusher := controller.New(
		processor.NewFactory(
			// TODO: Investigate
			// simple.NewWithHistogramDistribution(histogram.WithExplicitBoundaries([]float64{5, 10, 15})),
			simple.NewWithHistogramDistribution(histogram.WithExplicitBoundaries([]float64{5, 10, 15})),
			exp,
		),
		controller.WithExporter(exp),
		controller.WithCollectPeriod(1*time.Second),
	)

	global.SetMeterProvider(pusher)

	err := pusher.Start(ctx)
	handleErr(err, "Failed to start metric pusher", logger)

	return func() {
		cxt, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		// pushes any last exports to the receiver
		if err := pusher.Stop(cxt); err != nil {
			otel.Handle(err)
		}
	}
}

func handleErr(err error, message string, logger *zap.Logger) {
	if err != nil {
		logger.Fatal(message, zap.Error(err))
	}
}
