package cli

import (
	"github.com/urfave/cli/v2"
)

func New(version string) *cli.App {
	flags := getGlobalFlags()

	app := &cli.App{
		Name:    "otelgen",
		Usage:   "A tool to generate synthetic OpenTelemetry logs, metrics and traces",
		Version: version,
		Flags:   flags,
		Commands: []*cli.Command{
			genLogsCommand(),
			genMetricsCommand(),
			genTracesCommand(),
		},
	}

	app.EnableBashCompletion = true

	return app
}
