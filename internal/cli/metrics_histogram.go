package cli

import (
	"errors"

	"github.com/medxops/trazr-gen/internal/metrics"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

var generateMetricsHistogramCommand = &cli.Command{
	Name:        "histogram",
	Usage:       "generate metrics of type histogram",
	Description: "Histogram demonstrates how to measure a distribution of values",
	Aliases:     []string{"hist"},
	Flags: append(CommonMetricFlags,
		&cli.StringFlag{
			Name:  "temporality",
			Usage: "Temporality defines the window that an aggregation was calculated over, one of: delta, cumulative",
			Value: "cumulative",
		},
		&cli.StringFlag{
			Name:  "unit",
			Usage: "Unit of measurement for the histogram",
			Value: "ms",
		},
		&cli.StringSliceFlag{
			Name:  "attribute",
			Usage: "Attributes to add to the histogram (format: key=value)",
		},
		&cli.Float64SliceFlag{
			Name:  "bounds",
			Usage: "Bucket boundaries for the histogram",
			Value: cli.NewFloat64Slice(1, 5, 10, 25, 50, 100, 250, 500, 1000),
		},
		&cli.BoolFlag{
			Name:  "record-minmax",
			Usage: "Record min and max values",
			Value: true,
		},
	),
	Action: generateMetricsHistogramAction,
}

func generateMetricsHistogramAction(c *cli.Context) error {
	if c.String("output") == "" {
		return errors.New("'output' must be set")
	}

	metricsCfg := BuildMetricsConfig(c)

	if metricsCfg.Output == "terminal" || metricsCfg.Output == "stdout" {
		return metrics.SimulateHistogram(noop.NewMeterProvider(), metrics.HistogramConfig{}, metricsCfg, logger)
	}

	provider, err := SetupMetricProvider(c, metricsCfg)
	if err != nil {
		return err
	}

	logger.Info("Starting metrics generation")

	temporality := metricdata.CumulativeTemporality
	if c.String("temporality") == "delta" {
		temporality = metricdata.DeltaTemporality
	}

	attributes, err := parseAttributes(c.StringSlice("attribute"))
	if err != nil {
		logger.Error("failed to parse attributes", zap.Error(err))
		return err
	}

	histogramConfig := metrics.HistogramConfig{
		Name:         metricsCfg.ServiceName + ".metrics.histogram",
		Description:  "Histogram demonstrates how to measure a distribution of values",
		Unit:         c.String("unit"),
		Attributes:   attributes,
		Temporality:  temporality,
		Bounds:       c.Float64Slice("bounds"),
		RecordMinMax: c.Bool("record-minmax"),
	}

	if err := metrics.SimulateHistogram(provider, histogramConfig, metricsCfg, logger); err != nil {
		logger.Error("metrics histogram simulation failed", zap.Error(err))
		return err
	}

	return nil
}
