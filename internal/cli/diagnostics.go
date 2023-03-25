package cli

import (
	"github.com/urfave/cli/v2"
)

func genDiagnosticsCommand() *cli.Command {
	return &cli.Command{
		Name:    "diagnostics",
		Usage:   "Run connection diagnostics",
		Aliases: []string{"diags"},
		Hidden:  true,
		Subcommands: []*cli.Command{
			{
				Name:    "network",
				Usage:   "diagnose a connection to your receiver",
				Aliases: []string{"net"},
				Action: func(c *cli.Context) error {

					logger.Info("Not yet implemented")

					return nil
				},
			},
		},
	}
}
