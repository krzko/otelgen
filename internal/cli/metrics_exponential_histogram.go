package cli

import (
	"errors"

	"github.com/medxops/trazr-gen/internal/metrics"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

var generateMetricsExponentialHistogramCommand = &cli.Command{
	Name:        "exponential-histogram",
	Usage:       "generate metrics of type exponential histogram",
	Description: "ExponentialHistogram demonstrates how to measure a distribution of values with high dynamic range",
	Aliases:     []string{"ehist"},
	Flags: append(CommonMetricFlags,
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
	),
	Action: generateMetricsExponentialHistogramAction,
}

func generateMetricsExponentialHistogramAction(c *cli.Context) error {
	if c.String("output") == "" {
		return errors.New("'output' must be set")
	}

	metricsCfg := BuildMetricsConfig(c)

	if metricsCfg.Output == "terminal" || metricsCfg.Output == "stdout" {
		return metrics.SimulateExponentialHistogram(noop.NewMeterProvider(), metrics.ExponentialHistogramConfig{}, metricsCfg, logger)
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
	}

	expHistConfig := metrics.ExponentialHistogramConfig{
		Name:          metricsCfg.ServiceName + ".metrics.exponential_histogram",
		Description:   "ExponentialHistogram demonstrates how to measure a distribution of values with high dynamic range",
		Unit:          c.String("unit"),
		Attributes:    attributes,
		Temporality:   temporality,
		Scale:         int32(c.Int("scale")), // #nosec G115 -- CLI input is validated; overflow not a concern
		MaxSize:       c.Float64("max-size"),
		RecordMinMax:  c.Bool("record-minmax"),
		ZeroThreshold: c.Float64("zero-threshold"),
	}

	if err := metrics.SimulateExponentialHistogram(provider, expHistConfig, metricsCfg, logger); err != nil {
		logger.Error("metrics exponential histogram simulation failed", zap.Error(err))
		return err
	}

	return nil
}
