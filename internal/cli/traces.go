package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
						Name:    "marshal",
						Aliases: []string{"m"},
						Usage:   "marshal trace context via HTTP headers",
						Value:   false,
					},
					&cli.StringFlag{
						Name:    "scenario",
						Aliases: []string{"s"},
						Usage:   "The trace scenario to simulate (basic, eventing, microservices, web_mobile)",
						Value:   "basic",
					},
				},
				Action: func(c *cli.Context) error {
					return generateTraces(c, true)
				},
			},
			{
				Name:    "multi",
				Usage:   "generate multiple traces",
				Aliases: []string{"m"},
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:    "scenarios",
						Aliases: []string{"s"},
						Usage:   "The trace scenarios to simulate (basic, web_request, mobile_request, event_driven, pub_sub, microservices, database_operation)",
						Value:   cli.NewStringSlice("basic"),
					},
					&cli.IntFlag{
						Name:    "number-traces",
						Aliases: []string{"t"},
						Usage:   "number of traces to generate in each worker",
						Value:   3,
					},
					&cli.IntFlag{
						Name:    "workers",
						Aliases: []string{"w"},
						Usage:   "number of workers (goroutines) to run",
						Value:   1,
					},
				},
				Action: func(c *cli.Context) error {
					return generateTraces(c, false)
				},
			},
		},
	}
}

func generateTraces(c *cli.Context, isSingle bool) error {
	if c.String("otel-exporter-otlp-endpoint") == "" {
		return errors.New("'otel-exporter-otlp-endpoint' must be set")
	}

	tracesCfg := &traces.Config{
		Endpoint:    c.String("otel-exporter-otlp-endpoint"),
		ServiceName: c.String("service-name"),
		Insecure:    c.Bool("insecure"),
		UseHTTP:     c.String("protocol") == "http",
	}

	if isSingle {
		tracesCfg.NumTraces = 1
		tracesCfg.WorkerCount = 1
		tracesCfg.Scenarios = []string{c.String("scenario")}
		tracesCfg.PropagateContext = c.Bool("marshal")
	} else {
		tracesCfg.TotalDuration = time.Duration(c.Int("duration") * int(time.Second))
		tracesCfg.Rate = c.Int64("rate")
		tracesCfg.NumTraces = c.Int("number-traces")
		tracesCfg.WorkerCount = c.Int("workers")
		tracesCfg.Scenarios = c.StringSlice("scenarios")
		tracesCfg.PropagateContext = c.Bool("marshal")
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

	if tracesCfg.Insecure {
		grpcExpOpt = append(grpcExpOpt, otlptracegrpc.WithInsecure())
		httpExpOpt = append(httpExpOpt, otlptracehttp.WithInsecure())
	}

	if len(c.StringSlice("header")) > 0 {
		headers := make(map[string]string)
		for _, h := range c.StringSlice("header") {
			kv := strings.SplitN(h, "=", 2)
			if len(kv) != 2 {
				return fmt.Errorf("value should be of the format key=value")
			}
			headers[kv[0]] = kv[1]
		}
		grpcExpOpt = append(grpcExpOpt, otlptracegrpc.WithHeaders(headers))
		httpExpOpt = append(httpExpOpt, otlptracehttp.WithHeaders(headers))
		tracesCfg.Headers = headers
	}

	var exp *otlptrace.Exporter
	var err error
	if tracesCfg.UseHTTP {
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
		}
	}()

	ssp := sdktrace.NewBatchSpanProcessor(exp, sdktrace.WithBatchTimeout(time.Second))
	defer func() {
		logger.Info("stop the batch span processor")
		if err := ssp.Shutdown(context.Background()); err != nil {
			logger.Error("failed to stop the batch span processor", zap.Error(err))
		}
	}()

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(tracesCfg.ServiceName))),
		sdktrace.WithSpanProcessor(ssp),
	)

	otel.SetTracerProvider(tracerProvider)

	if err := traces.Run(tracesCfg, logger); err != nil {
		logger.Error("failed to run traces", zap.Error(err))
	}

	return nil
}
