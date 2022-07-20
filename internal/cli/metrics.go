package cli

import (
	"github.com/urfave/cli/v2"
)

func genMetricsCommand() *cli.Command {
	return &cli.Command{
		Name:    "metrics",
		Usage:   "Generate metrics",
		Aliases: []string{"m"},
		Subcommands: []*cli.Command{
			{
				Name:    "sum",
				Usage:   "generate metrics of type sum",
				Aliases: []string{"s"},
				// Action:  foo.Bar,
			},
		},
	}
}
