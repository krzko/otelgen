package cli

import (
	"context"
	"errors"
	"strings"
	"time"

	grpcZap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"

	"github.com/medxops/trazr-gen/internal/traces"

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
					&cli.StringFlag{
						Name:  "output",
						Usage: "OTLP output for traces export (or 'terminal' for stdout output)",
						Value: "terminal",
					},
					&cli.StringSliceFlag{
						Name:  "header",
						Usage: "Headers to send with OTLP requests (format: key=value)",
					},
					&cli.StringFlag{
						Name:  "service-name",
						Usage: "Service name for traces",
					},
					&cli.StringFlag{
						Name:  "protocol",
						Usage: "Protocol to use (grpc or http)",
					},
					&cli.BoolFlag{
						Name:  "insecure",
						Usage: "Use insecure connection",
					},
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
					&cli.StringSliceFlag{
						Name:    "attributes",
						Aliases: []string{"a"},
						Usage:   "Special attributes to inject into generated data (e.g., sensitive)",
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
					&cli.StringFlag{
						Name:  "output",
						Usage: "OTLP output for traces export (or 'terminal' for stdout output)",
						Value: "terminal",
					},
					&cli.StringSliceFlag{
						Name:  "header",
						Usage: "Headers to send with OTLP requests (format: key=value)",
					},
					&cli.StringFlag{
						Name:  "service-name",
						Usage: "Service name for traces",
					},
					&cli.StringFlag{
						Name:  "protocol",
						Usage: "Protocol to use (grpc or http)",
					},
					&cli.BoolFlag{
						Name:  "insecure",
						Usage: "Use insecure connection",
					},
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
						Name:  "duration",
						Usage: "duration in seconds for how long to generate traces",
					},
					&cli.Float64Flag{
						Name:  "rate",
						Usage: "rate of trace generation (per second)",
					},
					&cli.BoolFlag{
						Name:  "marshal",
						Usage: "marshal trace context via HTTP headers",
					},
					&cli.StringSliceFlag{
						Name:    "attributes",
						Aliases: []string{"a"},
						Usage:   "Special attributes to inject into generated data (e.g., sensitive)",
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
	if c.String("output") == "" {
		return errors.New("'output' must be set")
	}

	tracesCfg := &traces.Config{
		Output:      c.String("output"),
		ServiceName: c.String("service-name"),
		Insecure:    c.Bool("insecure"),
		UseHTTP:     c.String("protocol") == "http",
	}

	if isSingle {
		tracesCfg.NumTraces = 1
		tracesCfg.Scenarios = []string{c.String("scenario")}
		tracesCfg.PropagateContext = c.Bool("marshal")
	} else {
		tracesCfg.TotalDuration = time.Duration(c.Int("duration") * int(time.Second))
		tracesCfg.Rate = c.Float64("rate")
		tracesCfg.NumTraces = c.Int("number-traces")
		tracesCfg.Scenarios = c.StringSlice("scenarios")
		tracesCfg.PropagateContext = c.Bool("marshal")
	}

	// Add attributes from CLI
	tracesCfg.Attributes = c.StringSlice("attributes")

	if c.String("log-level") == "debug" {
		grpcZap.ReplaceGrpcLoggerV2(logger.WithOptions(
			zap.AddCallerSkip(3),
		))
	}

	var spanExporter sdktrace.SpanExporter
	var err error
	if tracesCfg.Output == "stdout" || tracesCfg.Output == "terminal" {
		spanExporter = &traces.StdoutSpanExporter{}
	} else {
		grpcExpOpt := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(tracesCfg.Output),
		}
		httpExpOpt := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(tracesCfg.Output),
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
					return errors.New("value should be of the format key=value")
				}
				headers[kv[0]] = kv[1]
			}
			grpcExpOpt = append(grpcExpOpt, otlptracegrpc.WithHeaders(headers))
			httpExpOpt = append(httpExpOpt, otlptracehttp.WithHeaders(headers))
			tracesCfg.Headers = headers
		}
		if tracesCfg.UseHTTP {
			logger.Info("starting HTTP exporter")
			spanExporter, err = otlptracehttp.New(context.Background(), httpExpOpt...)
		} else {
			logger.Info("starting gRPC exporter")
			spanExporter, err = otlptracegrpc.New(context.Background(), grpcExpOpt...)
		}
		if err != nil {
			logger.Error("failed to obtain OTLP exporter", zap.Error(err))
			return err
		}
	}
	defer func() {
		if spanExporter != nil {
			_ = spanExporter.Shutdown(context.Background())
		}
	}()

	ssp := sdktrace.NewBatchSpanProcessor(spanExporter, sdktrace.WithBatchTimeout(time.Second))
	defer func() {
		_ = ssp.Shutdown(context.Background())
	}()

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(tracesCfg.ServiceName))),
		sdktrace.WithSpanProcessor(ssp),
	)

	otel.SetTracerProvider(tracerProvider)

	if err := traces.Run(tracesCfg, logger); err != nil {
		logger.Error("failed to run traces", zap.Error(err))
		return err
	}

	return nil
}
