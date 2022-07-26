package cli

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var logger *zap.Logger

func initLogger(c *cli.Context) error {

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
		panic(err)
	}

	defer logger.Sync() // nolint: errcheck

	return err
}

func New(version, commit, date string) *cli.App {
	// Rainbow
	c := []color.Attribute{color.FgRed, color.FgGreen, color.FgYellow, color.FgMagenta, color.FgCyan, color.FgWhite, color.FgHiRed, color.FgHiGreen, color.FgHiYellow, color.FgHiBlue, color.FgHiMagenta, color.FgHiCyan, color.FgHiWhite}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(c), func(i, j int) { c[i], c[j] = c[j], c[i] })
	c0 := color.New(c[0]).SprintFunc()
	c1 := color.New(c[1]).SprintFunc()
	c2 := color.New(c[2]).SprintFunc()
	c3 := color.New(c[3]).SprintFunc()
	c4 := color.New(c[4]).SprintFunc()
	c5 := color.New(c[5]).SprintFunc()
	c6 := color.New(c[6]).SprintFunc()
	name := fmt.Sprintf("%s%s%s%s%s%s%s", c0("o"), c1("t"), c2("e"), c3("l"), c4("g"), c5("e"), c6("n"))

	flags := getGlobalFlags()

	v := fmt.Sprintf("v%v-%v (%v)", version, commit, date)
	app := &cli.App{
		Name:    name,
		Usage:   "A tool to generate synthetic OpenTelemetry logs, metrics and traces",
		Version: v,
		Flags:   flags,
		Commands: []*cli.Command{
			genDiagnosticsCommand(),
			genLogsCommand(),
			genMetricsCommand(),
			genTracesCommand(),
		},
		Before: initLogger,
	}

	app.EnableBashCompletion = true

	return app
}
