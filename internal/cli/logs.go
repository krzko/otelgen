package cli

import (
	"fmt"

	"github.com/krzko/otelgen/internal/logs"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func genLogsCommand() *cli.Command {
	return &cli.Command{
		Name:    "logs",
		Usage:   "Generate logs",
		Aliases: []string{"l"},
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "number",
				Aliases: []string{"n"},
				Usage:   "number of log events to generate",
				Value:   10,
			},
			&cli.IntFlag{
				Name:    "workers",
				Aliases: []string{"w"},
				Usage:   "number of workers (goroutines) to run",
				Value:   1,
			},
			&cli.StringFlag{
				Name:    "severity-text",
				Aliases: []string{"st"},
				Usage:   "Severity text of the log (e.g., Trace, Debug, Info, Warn, Error, Fatal)",
				Value:   "Info",
			},
			&cli.IntFlag{
				Name:    "severity-number",
				Aliases: []string{"sn"},
				Usage:   "Severity number of the log, range from 1 to 24 (inclusive)",
				Value:   9,
			},
		},
		Action: func(c *cli.Context) error {
			return generateLogs(c)
		},
	}
}

func generateLogs(c *cli.Context) error {
	logsCfg := &logs.Config{
		WorkerCount:    c.Int("workers"),
		NumLogs:        c.Int("number"),
		ServiceName:    c.String("service-name"),
		Endpoint:       c.String("otel-exporter-otlp-endpoint"),
		Insecure:       c.Bool("insecure"),
		UseHTTP:        c.String("protocol") == "http",
		Rate:           c.Float64("rate"),
		TotalDuration:  c.Duration("duration"),
		SeverityText:   c.String("severity-text"),
		SeverityNumber: int32(c.Int("severity-number")),
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	if err := logs.Run(logsCfg, logger); err != nil {
		logger.Error("failed to run logs generation", zap.Error(err))
		return err
	}

	return nil
}
