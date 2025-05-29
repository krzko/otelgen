//go:build !integration
// +build !integration

package cli

import (
	"testing"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func TestGenerateMetricsHistogramAction(t *testing.T) {
	origLogger := logger
	logger = zap.NewNop()
	defer func() { logger = origLogger }()

	// Happy path: valid config (should not error, but will run quickly)
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.IntFlag{Name: "duration"},
	}
	app.Commands = []*cli.Command{genMetricsCommand()}
	err := app.Run([]string{"trazr-gen", "metrics", "histogram", "--duration", "2", "--output", "terminal"})
	if err != nil {
		t.Errorf("expected no error for valid config, got %v", err)
	}
}
