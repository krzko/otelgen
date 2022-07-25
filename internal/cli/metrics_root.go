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
			generateMetricsCounterCommand,
			generateMetricsCounterObserverCommand,
			generateMetricsCounterObserverAdvancedCommand,
			generateMetricsCounterWithLabelsCommand,
			generateMetricsGaugeObserverCommand,
			generateMetricsHistogramCommand,
			generateMetricsUpDownCounterCommand,
			generateMetricsUpDownCounterObserverCommand,
		},
	}
}
