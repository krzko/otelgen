//go:build !integration
// +build !integration

package cli

import (
	"context"
	"testing"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func TestGenerateMetricsGaugeAction(t *testing.T) {
	origLogger := logger
	logger = zap.NewNop()
	defer func() { logger = origLogger }()

	// Happy path: valid config (should not error, but will run quickly)
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.IntFlag{Name: "duration"},
	}
	app.Commands = []*cli.Command{genMetricsCommand()}
	err := app.Run([]string{"trazr-gen", "metrics", "gauge", "--duration", "2", "--output", "terminal"})
	if err != nil {
		t.Errorf("expected no error for valid config, got %v", err)
	}
}

func TestMetricsGaugeCLIIntegration(t *testing.T) {
	origLogger := logger
	logger = zap.NewNop()
	defer func() { logger = origLogger }()

	// Use context with timeout to prevent hanging tests
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Happy path: valid config
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.IntFlag{Name: "duration"},
	}
	app.Commands = []*cli.Command{genMetricsCommand()}
	err := app.RunContext(ctx, []string{"trazr-gen", "metrics", "gauge", "--duration", "2", "--output", "terminal"})
	if err != nil {
		t.Errorf("expected no error for valid config, got %v", err)
	}
}
