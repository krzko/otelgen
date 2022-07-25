package metrics

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.uber.org/zap"
)

// Counter demonstrates how to measure non-decreasing numbers, for example,
// number of requests or connections.
func Counter(ctx context.Context, m metric.Meter, c *Config, logger *zap.Logger) {
	name := fmt.Sprintf("%v.metrics.counter", c.ServiceName)
	logger.Debug("generating counter", zap.String("name", name))
	counter, _ := m.SyncInt64().Counter(
		name,
		instrument.WithUnit("1"),
		instrument.WithDescription("Counter demonstrates how to measure non-decreasing numbers"),
	)

	var i int64
	if c.TotalDuration > 0 {
		logger.Info("generation duration", zap.Float64("seconds", c.TotalDuration.Seconds()))

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

// UpDownCounter demonstrates how to measure numbers that can go up and down, for example,
// number of goroutines or customers.
func UpDownCounter(ctx context.Context, m metric.Meter, c *Config, logger *zap.Logger) {
	name := fmt.Sprintf("%v.metrics.up_down_counter", c.ServiceName)
	counter, _ := m.SyncInt64().UpDownCounter(
		name,
		instrument.WithUnit("1"),
		instrument.WithDescription("UpDownCounter demonstrates how to measure numbers that can go up and down"),
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
			if rand.Float64() >= 0.5 {
				counter.Add(ctx, +1)
			} else {
				counter.Add(ctx, -1)
			}
			time.Sleep(time.Duration(c.Rate) * time.Second)
		}
	} else {
		for {
			logger.Info("generating", zap.String("name", name))
			if rand.Float64() >= 0.5 {
				counter.Add(ctx, +1)
			} else {
				counter.Add(ctx, -1)
			}
			time.Sleep(time.Duration(c.Rate) * time.Second)
		}
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
			time.Sleep(time.Duration(c.Rate) * time.Second)
		}
	} else {
		for {
			logger.Info("generating", zap.String("name", name))
			dur := time.Duration(rand.NormFloat64()*5000000) * time.Microsecond
			durRecorder.Record(ctx, dur.Microseconds())
			time.Sleep(time.Duration(c.Rate) * time.Second)
		}
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

	var number int64
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
			if err := m.RegisterCallback(
				[]instrument.Asynchronous{
					counter,
				},
				// SDK periodically calls this function to collect data.
				func(ctx context.Context) {
					number++
					counter.Observe(ctx, number)
					time.Sleep(time.Duration(c.Rate) * time.Second)
				},
			); err != nil {
				panic(err)
			}
		}
	} else {
		for {
			logger.Info("generating", zap.String("name", name))
			if err := m.RegisterCallback(
				[]instrument.Asynchronous{
					counter,
				},
				// SDK periodically calls this function to collect data.
				func(ctx context.Context) {
					number++
					counter.Observe(ctx, number)
					time.Sleep(time.Duration(c.Rate) * time.Second)
				},
			); err != nil {
				panic(err)
			}
		}
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
			if err := m.RegisterCallback(
				[]instrument.Asynchronous{
					counter,
				},
				func(ctx context.Context) {
					num := runtime.NumGoroutine()
					counter.Observe(ctx, int64(num))
					time.Sleep(time.Duration(c.Rate) * time.Second)
				},
			); err != nil {
				panic(err)
			}
		}
	} else {
		for {
			logger.Info("generating", zap.String("name", name))
			if err := m.RegisterCallback(
				[]instrument.Asynchronous{
					counter,
				},
				func(ctx context.Context) {
					num := runtime.NumGoroutine()
					counter.Observe(ctx, int64(num))
					time.Sleep(time.Duration(c.Rate) * time.Second)
				},
			); err != nil {
				panic(err)
			}
		}
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
	} else {
		for {
			logger.Info("generating", zap.String("name", name))
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
			if rand.Float64() < 0.3 {
				// increment hits
				counter.Add(ctx, 1, attribute.String("type", "hits"))
			} else {
				// increments misses
				counter.Add(ctx, 1, attribute.String("type", "misses"))
			}
			time.Sleep(time.Duration(c.Rate) * time.Second)
		}
	} else {
		for {
			if rand.Float64() < 0.3 {
				// increment hits
				counter.Add(ctx, 1, attribute.String("type", "hits"))
			} else {
				// increments misses
				counter.Add(ctx, 1, attribute.String("type", "misses"))
			}
			time.Sleep(time.Duration(c.Rate) * time.Second)
		}
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
			if rand.Float64() < 0.3 {
				atomic.AddInt64(&stats.Misses, 1)
			} else {
				atomic.AddInt64(&stats.Hits, 1)
			}
			time.Sleep(time.Duration(c.Rate) * time.Second)
		}
	} else {
		for {
			logger.Info("generating", zap.String("name", name))
			if rand.Float64() < 0.3 {
				atomic.AddInt64(&stats.Misses, 1)
			} else {
				atomic.AddInt64(&stats.Hits, 1)
			}
			time.Sleep(time.Duration(c.Rate) * time.Second)
		}
	}
}
