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

var generateMetricsGaugeCommand = &cli.Command{
	Name:        "gauge",
	Usage:       "generate metrics of type gauge",
	Description: "Gauge demonstrates how to measure a value that can go up and down",
	Aliases:     []string{"g"},
	Flags: []cli.Flag{
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
	},
	Action: func(c *cli.Context) error {
		return generateMetricsGaugeAction(c)
	},
}

func generateMetricsGaugeAction(c *cli.Context) error {
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

	metrics.SimulateGauge(provider, gaugeConfig, metricsCfg, logger)

	return nil
}
