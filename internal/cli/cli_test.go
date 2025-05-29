package cli

import (
	"context"
	"testing"
	"time"

	"github.com/urfave/cli/v2"
)

func TestInitLogger_DebugLevel(t *testing.T) {
	// Use context with timeout to prevent hanging tests
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "log-level"},
	}
	app.Before = initLogger
	app.Action = func(_ *cli.Context) error {
		if logger == nil {
			t.Error("logger should be initialized")
		}
		return nil
	}
	if err := app.RunContext(ctx, []string{"test", "--log-level", "debug"}); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestInitLogger_DefaultLevel(t *testing.T) {
	// Use context with timeout to prevent hanging tests
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "log-level"},
	}
	app.Before = initLogger
	app.Action = func(_ *cli.Context) error {
		if logger == nil {
			t.Error("logger should be initialized")
		}
		return nil
	}
	if err := app.RunContext(ctx, []string{"test"}); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestNewAppStructure(t *testing.T) {
	app := New("1.0.0", "abc123", "2024-01-01")
	if app.Name == "" {
		t.Error("expected app name to be set")
	}
	if app.Usage == "" {
		t.Error("expected app usage to be set")
	}
	if app.Version == "" {
		t.Error("expected app version to be set")
	}
	if len(app.Commands) < 1 {
		t.Error("expected at least one command")
	}
	if app.Before == nil {
		t.Error("expected Before hook to be set")
	}
}
