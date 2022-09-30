package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	grpcZap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/krzko/otelgen/internal/metrics"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/metric/export/aggregation"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var generateMetricsUpDownCounterCommand = &cli.Command{
	Name:        "up-down-counter",
	Usage:       "generate metrics of type up down counter",
	Description: "UpDownCounter demonstrates how to measure numbers that can go up and down",
	Aliases:     []string{"udc"},
	Action: func(c *cli.Context) error {
		return generateMetricsUpDownCounterAction(c)
	},
}

func generateMetricsUpDownCounterAction(c *cli.Context) error {
	var err error

	if c.String("otel-exporter-otlp-endpoint") == "" {
		return errors.New("'otel-exporter-otlp-endpoint' must be set")
	}

	metricsCfg := &metrics.Config{
		TotalDuration: time.Duration(c.Int("duration") * int(time.Second)),
		Endpoint:      c.String("otel-exporter-otlp-endpoint"),
		Rate:          c.Int64("rate"),
		ServiceName:   c.String("service-name"),
	}

	if c.String("log-level") == "debug" {
		grpcZap.ReplaceGrpcLoggerV2(logger.WithOptions(
			zap.AddCallerSkip(3),
		))
	}

	grpcExpOpt := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(metricsCfg.Endpoint),
		otlpmetricgrpc.WithDialOption(
			grpc.WithBlock(),
		),
	}

	httpExpOpt := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(metricsCfg.Endpoint),
	}

	if c.Bool("insecure") {
		grpcExpOpt = append(grpcExpOpt, otlpmetricgrpc.WithInsecure())
		httpExpOpt = append(httpExpOpt, otlpmetrichttp.WithInsecure())
	}

	if len(c.StringSlice("header")) > 0 {
		headers := make(map[string]string)
		logger.Debug("Header count", zap.Int("headers", len(c.StringSlice("header"))))
		for _, h := range c.StringSlice("header") {
			kv := strings.SplitN(h, "=", 2)
			if len(kv) != 2 {
				return fmt.Errorf("value should be of the format key=value")
			}
			logger.Debug("key=value", zap.String(kv[0], kv[1]))
			(headers)[kv[0]] = kv[1]

		}
		grpcExpOpt = append(grpcExpOpt, otlpmetricgrpc.WithHeaders(headers))
		httpExpOpt = append(httpExpOpt, otlpmetrichttp.WithHeaders(headers))
	}

	temporality := aggregation.CumulativeTemporalitySelector()
	if c.Bool("delta-temporality") {
		temporality = aggregation.DeltaTemporalitySelector()
	}

	var exp *otlpmetric.Exporter
	if c.String("protocol") == "http" {
		logger.Info("starting HTTP exporter")
		client := otlpmetrichttp.NewClient(httpExpOpt...)
		exp, err = otlpmetric.New(context.Background(), client, otlpmetric.WithMetricAggregationTemporalitySelector(temporality))
	} else {
		logger.Info("starting gRPC exporter")
		client := otlpmetricgrpc.NewClient(grpcExpOpt...)
		exp, err = otlpmetric.New(context.Background(), client, otlpmetric.WithMetricAggregationTemporalitySelector(temporality))
	}

	if err != nil {
		logger.Error("failed to obtain OTLP exporter", zap.Error(err))
		return err
	}
	defer func() {
		logger.Info("stopping the exporter")
		if err = exp.Shutdown(context.Background()); err != nil {
			logger.Error("failed to stop the exporter", zap.Error(err))
			return
		}
	}()

	ctx := context.Background()
	logger.Info("Starting metrics generation")

	var meter = global.MeterProvider().Meter(c.String("service-name"))

	if _, err := metrics.Run(ctx, exp, meter, metricsCfg, logger); err != nil {
		logger.Error("failed to stop the exporter", zap.Error(err))
	}

	metrics.UpDownCounter(ctx, meter, metricsCfg, logger)

	return nil
}
