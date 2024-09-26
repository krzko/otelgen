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

var generateMetricsSumCommand = &cli.Command{
	Name:        "sum",
	Usage:       "generate metrics of type sum",
	Description: "Sum demonstrates how to measure additive values over time",
	Aliases:     []string{"s"},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "temporality",
			Usage: "Temporality defines the window that an aggregation was calculated over, one of: delta, cumulative",
			Value: "cumulative",
		},
		&cli.StringFlag{
			Name:  "unit",
			Usage: "Unit of measurement for the sum",
			Value: "1",
		},
		&cli.StringSliceFlag{
			Name:  "attribute",
			Usage: "Attributes to add to the sum (format: key=value)",
		},
		&cli.BoolFlag{
			Name:  "monotonic",
			Usage: "Whether the sum is monotonic (always increasing)",
			Value: true,
		},
	},
	Action: func(c *cli.Context) error {
		return generateMetricsSumAction(c)
	},
}

func generateMetricsSumAction(c *cli.Context) error {
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
		logger.Warn("Delta temporality for sum metrics may not be supported by all backends. Consider using cumulative.")
		temporality = metricdata.DeltaTemporality
	}

	attributes, err := parseAttributes(c.StringSlice("attribute"))
	if err != nil {
		logger.Error("failed to parse attributes", zap.Error(err))
		return err
	}

	sumConfig := metrics.SumConfig{
		Name:        metricsCfg.ServiceName + ".metrics.sum",
		Description: "Sum demonstrates how to measure additive values over time",
		Unit:        c.String("unit"),
		Attributes:  attributes,
		Temporality: temporality,
		IsMonotonic: c.Bool("monotonic"),
	}

	metrics.SimulateSum(provider, sumConfig, metricsCfg, logger)

	return nil
}
