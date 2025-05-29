package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/medxops/trazr-gen/internal/logs"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var customLoggerFactory = newCustomLogger

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
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "output",
						Usage: "OTLP output for logs export (or 'terminal' for stdout output)",
						Value: "terminal",
					},
					&cli.StringSliceFlag{
						Name:  "header",
						Usage: "Headers to send with OTLP requests (format: key=value)",
					},
					&cli.StringSliceFlag{
						Name:    "attributes",
						Aliases: []string{"a"},
						Usage:   "Special attributes to inject into generated logs (e.g., sensitive)",
					},
				},
				Action: func(c *cli.Context) error {
					return generateLogs(c, true)
				},
			},
			{
				Name:    "multi",
				Usage:   "generate multiple logs",
				Aliases: []string{"m"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "output",
						Usage: "OTLP output for logs export (or 'terminal' for stdout output)",
						Value: "terminal",
					},
					&cli.StringSliceFlag{
						Name:  "header",
						Usage: "Headers to send with OTLP requests (format: key=value)",
					},
					&cli.IntFlag{
						Name:    "number",
						Aliases: []string{"n"},
						Usage:   "number of log events to generate",
						Value:   0, // Default to 0, which means indefinite
					},
					&cli.IntFlag{
						Name:    "duration",
						Aliases: []string{"d"},
						Usage:   "duration in seconds for how long to generate logs",
					},
					&cli.StringSliceFlag{
						Name:    "attributes",
						Aliases: []string{"a"},
						Usage:   "Special attributes to inject into generated logs (e.g., sensitive)",
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
	if c.String("output") == "" {
		return errors.New("'output' must be set")
	}

	logsCfg := &logs.Config{
		Output:      c.String("output"),
		ServiceName: c.String("service-name"),
		Insecure:    c.Bool("insecure"),
		UseHTTP:     c.String("protocol") == "http",
	}

	// Handle single log generation
	if isSingle {
		logsCfg.NumLogs = 1
	} else {
		logsCfg.NumLogs = c.Int("number")
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
			return errors.New("header format must be 'key=value'")
		}
		headers[kv[0]] = kv[1]
	}
	logsCfg.Headers = headers

	// Add attributes from CLI
	logsCfg.Attributes = c.StringSlice("attributes")

	// Set up logger without stack trace for warnings
	customLogger, err := customLoggerFactory()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	// Run the log generation
	if err := logs.Run(logsCfg, customLogger); err != nil {
		customLogger.Error("logs generation failed", zap.Error(err))
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
