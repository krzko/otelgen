package cli

import (
	"context"
	"errors"
	"time"

	"github.com/krzko/otelgen/internal/metrics"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

var generateMetricsExponentialHistogramCommand = &cli.Command{
	Name:        "exponential-histogram",
	Usage:       "generate metrics of type exponential histogram",
	Description: "ExponentialHistogram demonstrates how to measure a distribution of values with high dynamic range",
	Aliases:     []string{"ehist"},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "temporality",
			Usage: "Temporality defines the window that an aggregation was calculated over, one of: delta, cumulative",
			Value: "cumulative",
		},
		&cli.StringFlag{
			Name:  "unit",
			Usage: "Unit of measurement for the exponential histogram",
			Value: "ms",
		},
		&cli.StringSliceFlag{
			Name:  "attribute",
			Usage: "Attributes to add to the exponential histogram (format: key=value)",
		},
		&cli.IntFlag{
			Name:  "scale",
			Usage: "Scale factor for the exponential histogram buckets",
			Value: 0,
		},
		&cli.Float64Flag{
			Name:  "max-size",
			Usage: "Maximum value to generate (used to determine the range of values)",
			Value: 1000,
		},
		&cli.BoolFlag{
			Name:  "record-minmax",
			Usage: "Record min and max values",
			Value: true,
		},
		&cli.Float64Flag{
			Name:  "zero-threshold",
			Usage: "Threshold for the zero bucket",
			Value: 1e-6,
		},
	},
	Action: func(c *cli.Context) error {
		return generateMetricsExponentialHistogramAction(c)
	},
}

func generateMetricsExponentialHistogramAction(c *cli.Context) error {
	if c.String("otel-exporter-otlp-endpoint") == "" {
		return errors.New("'otel-exporter-otlp-endpoint' must be set")
	}

	metricsCfg := &metrics.Config{
		TotalDuration: time.Duration(c.Int("duration") * int(time.Second)),
		Endpoint:      c.String("otel-exporter-otlp-endpoint"),
		Rate:          c.Int64("rate"),
		ServiceName:   c.String("service-name"),
	}

	configureLogging(c)

	grpcExpOpt, httpExpOpt := getExporterOptions(c, metricsCfg)

	ctx := context.Background()

	exp, err := createExporter(ctx, c, grpcExpOpt, httpExpOpt)
	if err != nil {
		logger.Error("failed to obtain OTLP exporter", zap.Error(err))
		return err
	}
	defer shutdownExporter(exp)

	logger.Info("Starting metrics generation")

	reader := metric.NewPeriodicReader(
		exp,
		metric.WithInterval(time.Duration(metricsCfg.Rate)*time.Second),
	)

	provider := createMeterProvider(reader, metricsCfg)

	temporality := metricdata.CumulativeTemporality
	if c.String("temporality") == "delta" {
		temporality = metricdata.DeltaTemporality
	}

	attributes, err := parseAttributes(c.StringSlice("attribute"))
	if err != nil {
		logger.Error("failed to parse attributes", zap.Error(err))
		return err
	}

	expHistConfig := metrics.ExponentialHistogramConfig{
		Name:          metricsCfg.ServiceName + ".metrics.exponential_histogram",
		Description:   "ExponentialHistogram demonstrates how to measure a distribution of values with high dynamic range",
		Unit:          c.String("unit"),
		Attributes:    attributes,
		Temporality:   temporality,
		Scale:         int32(c.Int("scale")),
		MaxSize:       c.Float64("max-size"),
		RecordMinMax:  c.Bool("record-minmax"),
		ZeroThreshold: c.Float64("zero-threshold"),
	}

	metrics.SimulateExponentialHistogram(provider, expHistConfig, metricsCfg, logger)

	return nil
}
