// Package cli provides command-line interface utilities and commands for the trazr-gen application.
package cli

import (
	"fmt"

	"github.com/medxops/trazr-gen/internal/attributes"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var logger *zap.Logger

func initLogger(c *cli.Context) error {
	// Load sensitive data config if provided
	if configPath := c.String("config"); configPath != "" {
		err := attributes.LoadSensitiveConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load sensitive config: %w", err)
		}
	}

	var cfg zap.Config
	var err error

	switch c.String("log-level") {
	case "debug":
		cfg = zap.NewDevelopmentConfig()
	default:
		cfg = zap.NewProductionConfig()
	}
	logger, err = cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to build zap logger: %w", err)
	}

	defer logger.Sync() //nolint:errcheck

	return err
}

// New creates a new CLI application with the provided version, commit, and date information.
func New(version, commit, date string) *cli.App {
	name := "trazr-gen"

	flags := getGlobalFlags()

	var v string
	if version == "" {
		v = "develop"
	} else {
		v = fmt.Sprintf("v%v-%v (%v)", version, commit, date)
	}

	app := &cli.App{
		Name:    name,
		Usage:   "A tool to generate synthetic OpenTelemetry logs, metrics and traces",
		Version: v,
		Flags:   flags,
		Commands: []*cli.Command{
			// genDiagnosticsCommand(),
			genLogsCommand(),
			genMetricsCommand(),
			genTracesCommand(),
		},
		Before: initLogger,
	}

	app.EnableBashCompletion = true

	return app
}
