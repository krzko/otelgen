package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	grpcZap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
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
					},
				},
				Action: func(c *cli.Context) error {
					if c.String("otel-exporter-otlp-endpoint") == "" {
						return errors.New("'otel-exporter-otlp-endpoint' must be set")
					}

					cfg := &traces.Config{
						Endpoint:    c.String("otel-exporter-otlp-endpoint"),
						NumTraces:   1,
						WorkerCount: 1,
						ServiceName: c.String("service-name"),
					}

					logger, err := zap.NewProduction()
					if err != nil {
						panic(fmt.Sprintf("failed to obtain logger: %v", err))
					}
					defer logger.Sync()

					grpcZap.ReplaceGrpcLoggerV2(logger.WithOptions(
						zap.AddCallerSkip(3),
					))

					grpcExpOpt := []otlptracegrpc.Option{
						otlptracegrpc.WithEndpoint(cfg.Endpoint),
						otlptracegrpc.WithDialOption(
							grpc.WithBlock(),
						),
					}

					httpExpOpt := []otlptracehttp.Option{
						otlptracehttp.WithEndpoint(cfg.Endpoint),
					}

					if c.Bool("insecure") {
						grpcExpOpt = append(grpcExpOpt, otlptracegrpc.WithInsecure())
						httpExpOpt = append(httpExpOpt, otlptracehttp.WithInsecure())
					}

					if len(cfg.Headers) > 0 {
						grpcExpOpt = append(grpcExpOpt, otlptracegrpc.WithHeaders(cfg.Headers))
						httpExpOpt = append(httpExpOpt, otlptracehttp.WithHeaders(cfg.Headers))
					}

					var exp *otlptrace.Exporter
					if c.String("protocl") == "http" {
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
						sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(cfg.ServiceName))),
					)

					tracerProvider.RegisterSpanProcessor(ssp)
					otel.SetTracerProvider(tracerProvider)

					if err := traces.Run(cfg, logger); err != nil {
						logger.Error("failed to stop the exporter", zap.Error(err))
					}

					return nil
				},
			},
			{
				Name:    "multi",
				Usage:   "generate a multiple traces",
				Aliases: []string{"m"},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "masrshal",
						Aliases: []string{"m"},
						Usage:   "marshal trace context via HTTP headers",
						Value:   false,
					},
					&cli.IntFlag{
						Name:    "rate",
						Aliases: []string{"r"},
						Usage:   "rate of traces per second. 0 means no throttling",
						Value:   0,
					},
					&cli.IntFlag{
						Name:    "traces",
						Aliases: []string{"t"},
						Usage:   "number of traces to generate in each worker",
						Value:   1,
					},
					&cli.IntFlag{
						Name:    "workers",
						Aliases: []string{"t"},
						Usage:   "number of workers (goroutines) to run",
						Value:   1,
					},
				},
				Action: func(c *cli.Context) error {
					cfg := &traces.Config{
						Endpoint:    c.String("otel-exporter-otlp-endpoint"),
						NumTraces:   1,
						WorkerCount: 1,
						ServiceName: c.String("service-name"),
					}

					logger, err := zap.NewProduction()
					if err != nil {
						panic(fmt.Sprintf("failed to obtain logger: %v", err))
					}
					defer logger.Sync()

					grpcZap.ReplaceGrpcLoggerV2(logger.WithOptions(
						zap.AddCallerSkip(3),
					))

					grpcExpOpt := []otlptracegrpc.Option{
						otlptracegrpc.WithEndpoint(cfg.Endpoint),
						otlptracegrpc.WithDialOption(
							grpc.WithBlock(),
						),
					}

					httpExpOpt := []otlptracehttp.Option{
						otlptracehttp.WithEndpoint(cfg.Endpoint),
					}

					if c.Bool("insecure") {
						grpcExpOpt = append(grpcExpOpt, otlptracegrpc.WithInsecure())
						httpExpOpt = append(httpExpOpt, otlptracehttp.WithInsecure())
					}

					if len(cfg.Headers) > 0 {
						grpcExpOpt = append(grpcExpOpt, otlptracegrpc.WithHeaders(cfg.Headers))
						httpExpOpt = append(httpExpOpt, otlptracehttp.WithHeaders(cfg.Headers))
					}

					var exp *otlptrace.Exporter
					if c.String("protocl") == "http" {
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
						sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(cfg.ServiceName))),
					)

					tracerProvider.RegisterSpanProcessor(ssp)
					otel.SetTracerProvider(tracerProvider)

					if err := traces.Run(cfg, logger); err != nil {
						logger.Error("failed to stop the exporter", zap.Error(err))
					}

					return nil
				},
			},
		},
	}
}
