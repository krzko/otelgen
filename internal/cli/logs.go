package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
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
				Action: func(c *cli.Context) error {
					logger, err := zap.NewProduction()
					if err != nil {
						panic(fmt.Sprintf("failed to obtain logger: %v", err))
					}
					defer logger.Sync()

					logger.Info("soon (tm)")

					return nil
				},
			},
		},
	}
}
