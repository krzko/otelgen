package cli

import (
	"context"
	"errors"
	"time"

	"github.com/krzko/otelgen/internal/metrics"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

var generateMetricsHistogramCommand = &cli.Command{
	Name:        "histogram",
	Usage:       "generate metrics of type histogram",
	Description: "Histogram demonstrates how to record a distribution of individual values",
	Aliases:     []string{"h"},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "temporality",
			Usage: "Temporality defines the window that an aggregation was calculated over, one of: delta, cumulative",
			Value: "delta",
		},
	},
	Action: func(c *cli.Context) error {
		return generateMetricsHistogramAction(c)
	},
}

func generateMetricsHistogramAction(c *cli.Context) error {
	var err error

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
		metric.WithInterval(time.Duration(metricsCfg.Rate)),
	)

	provider := createMeterProvider(reader, metricsCfg)

	metrics.SimulateHistogram(provider, metricsCfg, logger)

	return nil
}
