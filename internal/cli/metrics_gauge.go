package cli

import (
	"errors"

	"github.com/medxops/trazr-gen/internal/metrics"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

var generateMetricsGaugeCommand = &cli.Command{
	Name:        "gauge",
	Usage:       "generate metrics of type gauge",
	Description: "Gauge demonstrates how to measure a value that can go up and down",
	Aliases:     []string{"g"},
	Flags: append(CommonMetricFlags,
		&cli.StringFlag{
			Name:  "temporality",
			Usage: "Temporality defines the window that an aggregation was calculated over, one of: delta, cumulative",
			Value: "cumulative",
		},
		&cli.StringFlag{
			Name:  "unit",
			Usage: "Unit of measurement for the gauge",
			Value: "1",
		},
		&cli.StringSliceFlag{
			Name:  "attribute",
			Usage: "Attributes to add to the gauge (format: key=value)",
		},
		&cli.Float64Flag{
			Name:  "min",
			Usage: "Minimum value for the gauge",
			Value: 0,
		},
		&cli.Float64Flag{
			Name:  "max",
			Usage: "Maximum value for the gauge",
			Value: 100,
		},
	),
	Action: generateMetricsGaugeAction,
}

func generateMetricsGaugeAction(c *cli.Context) error {
	if c.String("output") == "" {
		return errors.New("'output' must be set")
	}

	metricsCfg := BuildMetricsConfig(c)

	if metricsCfg.Output == "terminal" || metricsCfg.Output == "stdout" {
		return metrics.SimulateGauge(noop.NewMeterProvider(), metrics.GaugeConfig{}, metricsCfg, logger)
	}

	provider, err := SetupMetricProvider(c, metricsCfg)
	if err != nil {
		return err
	}

	logger.Info("Starting metrics generation")

	temporality := metricdata.CumulativeTemporality
	if c.String("temporality") == "delta" {
		logger.Warn("Delta temporality for gauge metrics may not be supported by all backends. Consider using cumulative.")
		temporality = metricdata.DeltaTemporality
	}

	attributes, err := parseAttributes(c.StringSlice("attribute"))
	if err != nil {
		logger.Error("failed to parse attributes", zap.Error(err))
		return err
	}

	gaugeConfig := metrics.GaugeConfig{
		Name:        metricsCfg.ServiceName + ".metrics.gauge",
		Description: "Gauge demonstrates how to measure a value that can go up and down",
		Unit:        c.String("unit"),
		Attributes:  attributes,
		Min:         c.Float64("min"),
		Max:         c.Float64("max"),
		Temporality: temporality,
	}

	if err := metrics.SimulateGauge(provider, gaugeConfig, metricsCfg, logger); err != nil {
		logger.Error("metrics gauge simulation failed", zap.Error(err))
		return err
	}

	return nil
}
