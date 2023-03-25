package cli

import (
	"context"
	"fmt"
	"strings"

	grpcZap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/krzko/otelgen/internal/metrics"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func genMetricsCommand() *cli.Command {
	return &cli.Command{
		Name:    "metrics",
		Usage:   "Generate metrics",
		Aliases: []string{"m"},
		Subcommands: []*cli.Command{
			generateMetricsCounterCommand,
			generateMetricsHistogramCommand,
			generateMetricsUpDownCounterCommand,
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
func NewMetricExporter(ctx context.Context, protocol string, options interface{}) (MetricExporter, error) {
	switch protocol {
	case "grpc":
		grpcOptions, ok := options.([]otlpmetricgrpc.Option)
		if !ok {
			return nil, fmt.Errorf("invalid options for grpc protocol")
		}
		return otlpmetricgrpc.New(ctx, grpcOptions...)
	case "http":
		httpOptions, ok := options.([]otlpmetrichttp.Option)
		if !ok {
			return nil, fmt.Errorf("invalid options for http protocol")
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

	if c.String("protocol") == "http" {
		logger.Info("starting HTTP exporter")
		exp, err = NewMetricExporter(ctx, "grpc", httpExpOpt)
		if err != nil {
			logger.Fatal("failed to create HTTP exporter: %v", zap.Error(err))
		}
	} else {
		logger.Info("starting gRPC exporter")
		exp, err = NewMetricExporter(ctx, "grpc", grpcExpOpt)
		if err != nil {
			logger.Fatal("failed to create gRPC exporter: %v", zap.Error(err))
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
		otlpmetricgrpc.WithEndpoint(mc.Endpoint),
		otlpmetricgrpc.WithDialOption(
			grpc.WithBlock(),
		),
	}

	httpExpOpt := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(mc.Endpoint),
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

	if c.String("temporality") == "delta" {
		grpcExpOpt = append(grpcExpOpt, otlpmetricgrpc.WithTemporalitySelector(preferDeltaTemporalitySelector))
		httpExpOpt = append(httpExpOpt, otlpmetrichttp.WithTemporalitySelector(preferDeltaTemporalitySelector))
	} else {
		grpcExpOpt = append(grpcExpOpt, otlpmetricgrpc.WithTemporalitySelector(preferCumulativeTemporalitySelector))
		httpExpOpt = append(httpExpOpt, otlpmetrichttp.WithTemporalitySelector(preferCumulativeTemporalitySelector))
	}

	return grpcExpOpt, httpExpOpt
}

// parseHeaders parses the headers from the command line and returns a map of string
func parseHeaders(c *cli.Context) (map[string]string, error) {
	headers := make(map[string]string)
	if len(c.StringSlice("header")) > 0 {
		for _, h := range c.StringSlice("header") {
			kv := strings.SplitN(h, "=", 2)
			if len(kv) != 2 {
				return nil, fmt.Errorf("value should be of the format key=value")
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

// shutdownExporter shuts down the exporter
func shutdownExporter(exp MetricExporter) {
	defer func() {
		logger.Info("stopping the exporter")
		if err := exp.Shutdown(context.Background()); err != nil {
			logger.Error("failed to stop the exporter", zap.Error(err))
			return
		}
	}()
}
