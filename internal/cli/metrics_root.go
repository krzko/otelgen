package cli

import (
	"context"

	grpcZap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/krzko/otelgen/internal/metrics"
)

func genMetricsCommand() *cli.Command {
	return &cli.Command{
		Name:    "metrics",
		Usage:   "Generate metrics",
		Aliases: []string{"m"},
		Subcommands: []*cli.Command{
			generateMetricsCounterCommand,
			generateMetricsCounterObserverCommand,
			generateMetricsCounterObserverAdvancedCommand,
			generateMetricsCounterWithLabelsCommand,
			generateMetricsGaugeObserverCommand,
			generateMetricsHistogramCommand,
			generateMetricsUpDownCounterCommand,
			generateMetricsUpDownCounterObserverCommand,
		},
	}
}

func setMetricsExporter(c *cli.Context, mc metrics.Config, logger *zap.Logger) (*otlpmetric.Exporter, error) {
	var err error

	grpcZap.ReplaceGrpcLoggerV2(logger.WithOptions(
		zap.AddCallerSkip(3),
	))

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

	var exp *otlpmetric.Exporter
	if c.String("protocl") == "http" {
		logger.Info("starting HTTP exporter")
		exp, err = otlpmetrichttp.New(context.Background(), httpExpOpt...)
	} else {
		logger.Info("starting gRPC exporter")
		exp, err = otlpmetricgrpc.New(context.Background(), grpcExpOpt...)
	}

	if err != nil {
		logger.Error("failed to obtain OTLP exporter", zap.Error(err))
		return exp, err
	}
	defer func() {
		logger.Info("stopping the exporter")
		if err = exp.Shutdown(context.Background()); err != nil {
			logger.Error("failed to stop the exporter", zap.Error(err))
			return
		}
	}()

	return exp, nil
}
