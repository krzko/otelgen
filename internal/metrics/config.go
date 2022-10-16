package metrics

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
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

// Run executes the test scenario.
func Run(ctx context.Context, exp *otlpmetric.Exporter, m metric.Meter, c *Config, logger *zap.Logger) (func(), error) {
	limit := rate.Limit(c.Rate)
	if c.Rate == 0 {
		limit = rate.Inf //nolint
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
		controller.WithCollectPeriod(time.Duration(c.Rate)*time.Second),
		controller.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(c.ServiceName))),
	)

	global.SetMeterProvider(pusher)
	if err := pusher.Start(ctx); err != nil {
		logger.Error("Failed to start metric pusher", zap.Error(err))
	}

	return func() {
		cxt, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		// pushes any last exports to the receiver
		if err := pusher.Stop(cxt); err != nil {
			otel.Handle(err)
		}
	}, nil
}
