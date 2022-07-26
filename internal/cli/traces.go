package cli

import (
	"context"
	"errors"
	"time"

	grpcZap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"google.golang.org/grpc"

	"github.com/krzko/otelgen/internal/traces"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func genTracesCommand() *cli.Command {
	return &cli.Command{
		Name:    "traces",
		Usage:   "Generate traces",
		Aliases: []string{"t"},
		Subcommands: []*cli.Command{
			{
				Name:    "single",
				Usage:   "generate a single trace",
				Aliases: []string{"s"},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "masrshal",
						Aliases: []string{"m"},
						Usage:   "marshal trace context via HTTP headers",
						Value:   false,
						Hidden:  true,
					},
				},
				Action: func(c *cli.Context) error {
					var err error

					if c.String("otel-exporter-otlp-endpoint") == "" {
						return errors.New("'otel-exporter-otlp-endpoint' must be set")
					}

					tracesCfg := &traces.Config{
						Endpoint:    c.String("otel-exporter-otlp-endpoint"),
						NumTraces:   1,
						WorkerCount: 1,
						ServiceName: c.String("service-name"),
					}

					if c.String("log-level") == "debug" {
						grpcZap.ReplaceGrpcLoggerV2(logger.WithOptions(
							zap.AddCallerSkip(3),
						))
					}

					grpcExpOpt := []otlptracegrpc.Option{
						otlptracegrpc.WithEndpoint(tracesCfg.Endpoint),
						otlptracegrpc.WithDialOption(
							grpc.WithBlock(),
						),
					}

					httpExpOpt := []otlptracehttp.Option{
						otlptracehttp.WithEndpoint(tracesCfg.Endpoint),
					}

					if c.Bool("insecure") {
						grpcExpOpt = append(grpcExpOpt, otlptracegrpc.WithInsecure())
						httpExpOpt = append(httpExpOpt, otlptracehttp.WithInsecure())
					}

					if len(tracesCfg.Headers) > 0 {
						grpcExpOpt = append(grpcExpOpt, otlptracegrpc.WithHeaders(tracesCfg.Headers))
						httpExpOpt = append(httpExpOpt, otlptracehttp.WithHeaders(tracesCfg.Headers))
					}

					var exp *otlptrace.Exporter
					if c.String("protocol") == "http" {
						logger.Info("starting HTTP exporter")
						exp, err = otlptracehttp.New(context.Background(), httpExpOpt...)
					} else {
						logger.Info("starting gRPC exporter")
						exp, err = otlptracegrpc.New(context.Background(), grpcExpOpt...)
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

					ssp := sdktrace.NewBatchSpanProcessor(exp, sdktrace.WithBatchTimeout(time.Second))
					defer func() {
						logger.Info("stop the batch span processor")
						if err := ssp.Shutdown(context.Background()); err != nil {
							logger.Error("failed to stop the batch span processor", zap.Error(err))
							return
						}
					}()

					tracerProvider := sdktrace.NewTracerProvider(
						sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(tracesCfg.ServiceName))),
					)

					tracerProvider.RegisterSpanProcessor(ssp)
					otel.SetTracerProvider(tracerProvider)

					if err := traces.Run(tracesCfg, logger); err != nil {
						logger.Error("failed to stop the exporter", zap.Error(err))
					}

					return nil
				},
			},
			{
				Name:    "multi",
				Usage:   "generate multiple traces",
				Aliases: []string{"m"},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "masrshal",
						Aliases: []string{"m"},
						Usage:   "marshal trace context via HTTP headers",
						Value:   false,
						Hidden:  true,
					},
					&cli.IntFlag{
						Name:    "number-traces",
						Aliases: []string{"t"},
						Usage:   "number of traces to generate in each worker",
						Value:   10,
					},
					&cli.IntFlag{
						Name:    "workers",
						Aliases: []string{"w"},
						Usage:   "number of workers (goroutines) to run",
						Value:   1,
					},
				},
				Action: func(c *cli.Context) error {
					var err error
					defer logger.Sync()

					if c.String("otel-exporter-otlp-endpoint") == "" {
						return errors.New("'otel-exporter-otlp-endpoint' must be set")
					}

					tracesCfg := &traces.Config{
						TotalDuration: time.Duration(c.Int("duration") * int(time.Second)),
						Endpoint:      c.String("otel-exporter-otlp-endpoint"),
						Rate:          c.Int64("rate"),
						NumTraces:     c.Int("number-traces"),
						WorkerCount:   c.Int("workers"),
						ServiceName:   c.String("service-name"),
					}

					if c.String("log-level") == "debug" {
						grpcZap.ReplaceGrpcLoggerV2(logger.WithOptions(
							zap.AddCallerSkip(3),
						))
					}

					grpcExpOpt := []otlptracegrpc.Option{
						otlptracegrpc.WithEndpoint(tracesCfg.Endpoint),
						otlptracegrpc.WithDialOption(
							grpc.WithBlock(),
						),
					}

					httpExpOpt := []otlptracehttp.Option{
						otlptracehttp.WithEndpoint(tracesCfg.Endpoint),
					}

					if c.Bool("insecure") {
						grpcExpOpt = append(grpcExpOpt, otlptracegrpc.WithInsecure())
						httpExpOpt = append(httpExpOpt, otlptracehttp.WithInsecure())
					}

					if len(tracesCfg.Headers) > 0 {
						grpcExpOpt = append(grpcExpOpt, otlptracegrpc.WithHeaders(tracesCfg.Headers))
						httpExpOpt = append(httpExpOpt, otlptracehttp.WithHeaders(tracesCfg.Headers))
					}

					var exp *otlptrace.Exporter
					if c.String("protocol") == "http" {
						logger.Info("starting HTTP exporter")
						exp, err = otlptracehttp.New(context.Background(), httpExpOpt...)
					} else {
						logger.Info("starting gRPC exporter")
						exp, err = otlptracegrpc.New(context.Background(), grpcExpOpt...)
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

					ssp := sdktrace.NewBatchSpanProcessor(exp, sdktrace.WithBatchTimeout(time.Second))
					defer func() {
						logger.Info("stop the batch span processor")
						if err := ssp.Shutdown(context.Background()); err != nil {
							logger.Error("failed to stop the batch span processor", zap.Error(err))
							return
						}
					}()

					tracerProvider := sdktrace.NewTracerProvider(
						sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(tracesCfg.ServiceName))),
					)

					tracerProvider.RegisterSpanProcessor(ssp)
					otel.SetTracerProvider(tracerProvider)

					if err := traces.Run(tracesCfg, logger); err != nil {
						logger.Error("failed to stop the exporter", zap.Error(err))
					}

					return nil
				},
			},
		},
	}
}
