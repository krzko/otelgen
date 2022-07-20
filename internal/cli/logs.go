package cli

import (
	"github.com/urfave/cli/v2"
)

func genLogsCommand() *cli.Command {
	return &cli.Command{
		Name:    "logs",
		Usage:   "Generate logs",
		Aliases: []string{"l"},
		Subcommands: []*cli.Command{
			{
				Name:    "single",
				Usage:   "generate a single log entry",
				Aliases: []string{"s"},
				// Action:  foo.Bar,
			},
		},
	}
}
