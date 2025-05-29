package cli

import (
	"errors"

	"github.com/medxops/trazr-gen/internal/metrics"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

var generateMetricsSumCommand = &cli.Command{
	Name:        "sum",
	Usage:       "generate metrics of type sum",
	Description: "Sum demonstrates how to measure additive values over time",
	Aliases:     []string{"s"},
	Flags: append(CommonMetricFlags,
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
	),
	Action: generateMetricsSumAction,
}

func generateMetricsSumAction(c *cli.Context) error {
	if c.String("output") == "" {
		return errors.New("'output' must be set")
	}

	headers, err := parseHeaders(c)
	if err != nil {
		return err
	}

	metricsCfg := BuildMetricsConfig(c)
	metricsCfg.Headers = headers

	provider, err := SetupMetricProvider(c, metricsCfg)
	if err != nil {
		return err
	}

	logger.Info("Starting metrics generation")

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

	if metricsCfg.Rate <= 0 {
		return errors.New("rate must be positive for SimulateSum")
	}

	metrics.SimulateSum(provider, sumConfig, metricsCfg, logger)

	return nil
}
