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

func TestGenerateTraces_ValidMinimal(t *testing.T) {
	origLogger := logger
	logger = zap.NewNop()
	defer func() { logger = origLogger }()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	app := cli.NewApp()
	app.Commands = []*cli.Command{
		{
			Name:   "traces",
			Action: func(c *cli.Context) error { return generateTraces(c, false) },
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "output"},
				&cli.StringFlag{Name: "service-name", Value: "test-service"},
				&cli.IntFlag{Name: "number-traces", Value: 1},
				&cli.IntFlag{Name: "workers", Value: 1},
				&cli.StringFlag{Name: "protocol", Value: "http"},
			},
		},
	}
	err := app.RunContext(ctx, []string{"trazr-gen", "traces", "--output", "terminal"})
	// OpenTelemetry exporters do not error on unreachable endpoints at creation time;
	// errors are reported asynchronously during export. So we expect no error here.
	if err != nil {
		t.Errorf("expected no error for minimal valid config, got: %v", err)
	}
}
