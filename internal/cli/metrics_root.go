package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	grpcZap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/medxops/trazr-gen/internal/metrics"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

// CommonMetricFlags defines flags shared by all metric subcommands
var CommonMetricFlags = []cli.Flag{
	&cli.StringFlag{
		Name:  "output",
		Usage: "OTLP output for metrics export (or 'terminal' for stdout output)",
		Value: "terminal",
	},
	&cli.StringSliceFlag{
		Name:  "header",
		Usage: "Headers to send with OTLP requests (format: key=value)",
	},
	&cli.IntFlag{
		Name:  "duration",
		Usage: "Duration in seconds for how long to generate metrics",
	},
	&cli.Float64Flag{
		Name:  "rate",
		Usage: "Number of events generated per second (0 = unthrottled)",
		Value: 1.0,
	},
	&cli.StringFlag{
		Name:  "service-name",
		Usage: "Service name to use",
		Value: "trazr-gen",
	},
}

// BuildMetricsConfig constructs a metrics.Config from the CLI context
func BuildMetricsConfig(c *cli.Context) *metrics.Config {
	rate := c.Float64("rate")
	if rate < 0 {
		rate = 1.0
	}
	return &metrics.Config{
		NumMetrics:    1,
		TotalDuration: time.Duration(c.Int("duration")) * time.Second,
		Output:        c.String("output"),
		Rate:          rate,
		ServiceName:   c.String("service-name"),
	}
}

func genMetricsCommand() *cli.Command {
	return &cli.Command{
		Name:    "metrics",
		Usage:   "Generate metrics",
		Aliases: []string{"m"},
		Subcommands: []*cli.Command{
			generateMetricsExponentialHistogramCommand,
			generateMetricsGaugeCommand,
			generateMetricsHistogramCommand,
			generateMetricsSumCommand,
		},
	}
}

// MetricExporter is an interface that abstracts the functionality of both
// otlpmetricgrpc and otlpmetrichttp exporters.
type MetricExporter interface {
	metric.Exporter
}

// NewMetricExporter creates a new MetricExporter based on the provided protocol.
// It accepts either otlpmetricgrpc.Option or otlpmetrichttp.Option as an argument.
// It returns an implementation of the MetricExporter interface.
func NewMetricExporter(ctx context.Context, protocol string, options any) (MetricExporter, error) {
	switch protocol {
	case "grpc":
		grpcOptions, ok := options.([]otlpmetricgrpc.Option)
		if !ok {
			return nil, errors.New("invalid options for grpc protocol")
		}
		return otlpmetricgrpc.New(ctx, grpcOptions...)
	case "http":
		httpOptions, ok := options.([]otlpmetrichttp.Option)
		if !ok {
			return nil, errors.New("invalid options for http protocol")
		}
		return otlpmetrichttp.New(ctx, httpOptions...)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// configureLogging configures the logging level for the gRPC logger
func configureLogging(c *cli.Context) {
	if c.String("log-level") == "debug" {
		grpcZap.ReplaceGrpcLoggerV2(logger.WithOptions(
			zap.AddCallerSkip(3),
		))
	}
}

// createExporter creates a new exporter based on the command line flags
func createExporter(ctx context.Context, c *cli.Context, grpcExpOpt []otlpmetricgrpc.Option, httpExpOpt []otlpmetrichttp.Option) (MetricExporter, error) {
	var exp MetricExporter
	var err error

	if c.String("output") == "terminal" || c.String("output") == "stdout" {
		exp = &metrics.StdoutMetricExporter{}
		return exp, nil
	}

	if c.String("protocol") == "http" {
		logger.Info("starting HTTP exporter")
		exp, err = NewMetricExporter(ctx, "http", httpExpOpt)
		if err != nil {
			logger.Error("failed to create HTTP exporter", zap.Error(err))
			return nil, err
		}
	} else {
		logger.Info("starting gRPC exporter")
		exp, err = NewMetricExporter(ctx, "grpc", grpcExpOpt)
		if err != nil {
			logger.Error("failed to create gRPC exporter", zap.Error(err))
			return nil, err
		}
	}

	return exp, err
}

// createReader creates a new reader based on the command line flags
func createMeterProvider(reader metric.Reader, metricsCfg *metrics.Config) *metric.MeterProvider {
	provider := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(metricsCfg.ServiceName),
			semconv.DeploymentEnvironment("local"),
		)),
	)

	return provider
}

// getExporterOptions returns the exporter options based on the command line flags
func getExporterOptions(c *cli.Context, mc *metrics.Config) ([]otlpmetricgrpc.Option, []otlpmetrichttp.Option) {
	grpcExpOpt := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(mc.Output),
	}

	httpExpOpt := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(mc.Output),
	}

	if c.Bool("insecure") {
		grpcExpOpt = append(grpcExpOpt, otlpmetricgrpc.WithInsecure())
		httpExpOpt = append(httpExpOpt, otlpmetrichttp.WithInsecure())
	}

	headers, _ := parseHeaders(c)
	if len(headers) > 0 {
		grpcExpOpt = append(grpcExpOpt, otlpmetricgrpc.WithHeaders(headers))
		httpExpOpt = append(httpExpOpt, otlpmetrichttp.WithHeaders(headers))
	}

	switch c.String("temporality") {
	case "delta":
		logger.Info("using", zap.String("temporarility", c.String("temporality")))
		grpcExpOpt = append(grpcExpOpt, otlpmetricgrpc.WithTemporalitySelector(preferDeltaTemporalitySelector))
		httpExpOpt = append(httpExpOpt, otlpmetrichttp.WithTemporalitySelector(preferDeltaTemporalitySelector))
	case "cumulative":
		logger.Info("using", zap.String("temporarility", c.String("temporality")))
		grpcExpOpt = append(grpcExpOpt, otlpmetricgrpc.WithTemporalitySelector(preferCumulativeTemporalitySelector))
		httpExpOpt = append(httpExpOpt, otlpmetrichttp.WithTemporalitySelector(preferCumulativeTemporalitySelector))
	default:
		logger.Error("falliing back to delta temporality", zap.String("use one of", "delta, cumulative"))
		grpcExpOpt = append(grpcExpOpt, otlpmetricgrpc.WithTemporalitySelector(preferDeltaTemporalitySelector))
		httpExpOpt = append(httpExpOpt, otlpmetrichttp.WithTemporalitySelector(preferDeltaTemporalitySelector))
	}

	return grpcExpOpt, httpExpOpt
}

// parseAttributes parses the attributes from the command line and returns a slice of attribute.KeyValue
func parseAttributes(attrs []string) ([]attribute.KeyValue, error) {
	var result []attribute.KeyValue
	for i, attr := range attrs {
		parts := strings.SplitN(attr, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid attribute format at index %d: %s (expected key=value)", i, attr)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("empty key in attribute at index %d: %s", i, attr)
		}
		result = append(result, attribute.String(key, value))
	}
	return result, nil
}

// parseHeaders parses the headers from the command line and returns a map of string
func parseHeaders(c *cli.Context) (map[string]string, error) {
	headers := make(map[string]string)
	if len(c.StringSlice("header")) > 0 {
		for _, h := range c.StringSlice("header") {
			kv := strings.SplitN(h, "=", 2)
			if len(kv) != 2 {
				return nil, errors.New("value should be of the format key=value")
			}
			headers[kv[0]] = kv[1]
		}
	}
	return headers, nil
}

// preferDeltaTemporalitySelector returns delta temporality for an instrument kind
func preferDeltaTemporalitySelector(kind metric.InstrumentKind) metricdata.Temporality {
	switch kind {
	case metric.InstrumentKindCounter,
		metric.InstrumentKindObservableCounter,
		metric.InstrumentKindUpDownCounter,
		metric.InstrumentKindHistogram:
		return metricdata.DeltaTemporality
	default:
		return metricdata.CumulativeTemporality
	}
}

// preferCumulativeTemporalitySelector returns cumulative temporality for an instrument kind
func preferCumulativeTemporalitySelector(kind metric.InstrumentKind) metricdata.Temporality {
	switch kind {
	case metric.InstrumentKindCounter,
		metric.InstrumentKindObservableCounter,
		metric.InstrumentKindUpDownCounter,
		metric.InstrumentKindHistogram:
		return metricdata.CumulativeTemporality
	default:
		return metricdata.DeltaTemporality
	}
}

// SetupMetricProvider centralizes exporter, reader, and provider setup for metrics CLI subcommands
func SetupMetricProvider(c *cli.Context, metricsCfg *metrics.Config) (*metric.MeterProvider, error) {
	configureLogging(c)
	grpcExpOpt, httpExpOpt := getExporterOptions(c, metricsCfg)
	ctx := context.Background()
	exp, err := createExporter(ctx, c, grpcExpOpt, httpExpOpt)
	if err != nil {
		logger.Error("failed to obtain OTLP exporter", zap.Error(err))
		return nil, err
	}
	reader := metric.NewPeriodicReader(
		exp,
		metric.WithInterval(time.Duration(metricsCfg.Rate)*time.Second),
	)
	provider := createMeterProvider(reader, metricsCfg)
	return provider, nil
}
