package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/krzko/otelgen/internal/logs"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func genLogsCommand() *cli.Command {
	return &cli.Command{
		Name:    "logs",
		Usage:   "Generate logs",
		Aliases: []string{"l"},
		Subcommands: []*cli.Command{
			{
				Name:    "single",
				Usage:   "generate a single log event",
				Aliases: []string{"s"},
				Action: func(c *cli.Context) error {
					return generateLogs(c, true)
				},
			},
			{
				Name:    "multi",
				Usage:   "generate multiple logs",
				Aliases: []string{"m"},
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "number",
						Aliases: []string{"n"},
						Usage:   "number of log events to generate",
						Value:   0, // Default to 0, which means indefinite
					},
					&cli.IntFlag{
						Name:    "workers",
						Aliases: []string{"w"},
						Usage:   "number of workers (goroutines) to run",
						Value:   1,
					},
					&cli.IntFlag{
						Name:    "duration",
						Aliases: []string{"d"},
						Usage:   "duration in seconds for how long to generate logs",
					},
				},
				Action: func(c *cli.Context) error {
					return generateLogs(c, false)
				},
			},
		},
	}
}

func generateLogs(c *cli.Context, isSingle bool) error {
	if c.String("otel-exporter-otlp-endpoint") == "" {
		return errors.New("'otel-exporter-otlp-endpoint' must be set")
	}

	logsCfg := &logs.Config{
		Endpoint:    c.String("otel-exporter-otlp-endpoint"),
		ServiceName: c.String("service-name"),
		Insecure:    c.Bool("insecure"),
		UseHTTP:     c.String("protocol") == "http",
	}

	// Handle single log generation
	if isSingle {
		logsCfg.NumLogs = 1
		logsCfg.WorkerCount = 1
	} else {
		logsCfg.NumLogs = c.Int("number")
		logsCfg.WorkerCount = c.Int("workers")
		logsCfg.TotalDuration = time.Duration(c.Int("duration") * int(time.Second))
		logsCfg.Rate = c.Float64("rate")

		// If neither `NumLogs` nor `TotalDuration` is set, default to indefinite generation
		if logsCfg.NumLogs == 0 && logsCfg.TotalDuration == 0 {
			logsCfg.NumLogs = 0 // Indefinite
			logsCfg.TotalDuration = 0
		}
	}

	// Parse headers
	headers := make(map[string]string)
	for _, h := range c.StringSlice("header") {
		kv := strings.SplitN(h, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("header format must be 'key=value'")
		}
		headers[kv[0]] = kv[1]
	}
	logsCfg.Headers = headers

	// Set up logger without stack trace for warnings
	logger, err := newCustomLogger()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	// Run the log generation
	if err := logs.Run(logsCfg, logger); err != nil {
		logger.Error("failed to run logs generation", zap.Error(err))
		return err
	}

	return nil
}

func newCustomLogger() (*zap.Logger, error) {
	cfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.DebugLevel),
		Development: true,
		Sampling:    nil,
		Encoding:    "json", // or "console" if you prefer
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:    "message",
			LevelKey:      "level",
			TimeKey:       "time",
			NameKey:       "logger",
			CallerKey:     "caller",
			StacktraceKey: "stacktrace", // This will hold stack trace information
			LineEnding:    zapcore.DefaultLineEnding,
			EncodeLevel:   zapcore.CapitalLevelEncoder,
			EncodeTime:    zapcore.ISO8601TimeEncoder,
			EncodeCaller:  zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// Disable stacktrace for warnings and below
	cfg.EncoderConfig.StacktraceKey = ""
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)

	return cfg.Build()
}
