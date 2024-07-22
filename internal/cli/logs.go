package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	grpcZap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"google.golang.org/grpc"

	"github.com/krzko/otelgen/internal/logs"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func genLogsCommand() *cli.Command {
	return &cli.Command{
		Name:    "logs",
		Usage:   "Generate logs",
		Aliases: []string{"l"},
		Subcommands: []*cli.Command{
			{
				Name:    "single",
				Usage:   "generate a single log entry",
				Aliases: []string{"s"},
				Action: func(c *cli.Context) error {
					if c.String("otel-exporter-otlp-endpoint") == "" {
						return errors.New("'otel-exporter-otlp-endpoint' must be set")
					}

					logsCfg := &logs.Config{
						Endpoint: c.String("otel-exporter-otlp-endpoint"),
						NumLogs:  1,
						// WorkerCount: 1,
						ServiceName: c.String("service-name"),
					}

					if c.String("log-level") == "debug" {
						grpcZap.ReplaceGrpcLoggerV2(logger.WithOptions(
							zap.AddCallerSkip(3),
						))
					}

					grpcExpOpt := []otlploggrpc.Option{
						otlploggrpc.WithEndpoint(logsCfg.Endpoint),
						otlploggrpc.WithDialOption(
							grpc.WithBlock(),
						),
					}

					httpExpOpt := []otlploghttp.Option{
						otlploghttp.WithEndpoint(logsCfg.Endpoint),
					}

					if c.Bool("insecure") {
						grpcExpOpt = append(grpcExpOpt, otlploggrpc.WithInsecure())
						httpExpOpt = append(httpExpOpt, otlploghttp.WithInsecure())
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
						grpcExpOpt = append(grpcExpOpt, otlploggrpc.WithHeaders(headers))
						httpExpOpt = append(httpExpOpt, otlploghttp.WithHeaders(headers))
					}

					var blp *sdklog.BatchProcessor
					if c.String("protocol") == "http" {
						logger.Info("starting HTTP exporter")
						exp, err := otlploghttp.New(context.Background(), httpExpOpt...)
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
						blp = sdklog.NewBatchProcessor(exp, sdklog.WithExportTimeout(time.Second))
					} else {
						logger.Info("starting gRPC exporter")
						exp, err := otlploggrpc.New(context.Background(), grpcExpOpt...)
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
						blp = sdklog.NewBatchProcessor(exp, sdklog.WithExportTimeout(time.Second))
					}

					defer func() {
						logger.Info("stop the batch log processor")
						if err := blp.Shutdown(context.Background()); err != nil {
							logger.Error("failed to stop the batch log processor", zap.Error(err))
							return
						}
					}()

					loggerProvider := sdklog.NewLoggerProvider(
						sdklog.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(logsCfg.ServiceName))),
						sdklog.WithProcessor(blp),
					)

					global.SetLoggerProvider(loggerProvider)

					if err := logs.Run(logsCfg); err != nil {
						logger.Error("failed to stop the exporter", zap.Error(err))
					}

					return nil
				},
			},
		},
	}
}
