//go:build !integration
// +build !integration

package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func TestGenLogsCommand_Structure(t *testing.T) {
	cmd := genLogsCommand()
	if cmd.Name != "logs" {
		t.Errorf("expected command name 'logs', got '%s'", cmd.Name)
	}
	if cmd.Usage == "" {
		t.Error("expected non-empty usage")
	}
	if len(cmd.Subcommands) != 2 {
		t.Errorf("expected 2 subcommands, got %d", len(cmd.Subcommands))
	}
	subNames := []string{cmd.Subcommands[0].Name, cmd.Subcommands[1].Name}
	if subNames[0] != "single" || subNames[1] != "multi" {
		t.Errorf("expected subcommands 'single' and 'multi', got %v", subNames)
	}
}

func TestGenerateLogs_InvalidHeaderFormat(t *testing.T) {
	// Use context with timeout to prevent hanging tests
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	app := cli.NewApp()
	app.Commands = []*cli.Command{genLogsCommand()}
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "output"},
		&cli.StringSliceFlag{Name: "header"},
		&cli.IntFlag{Name: "number"},
	}
	err := app.RunContext(ctx, []string{"trazr-gen", "logs", "multi", "--output", "foo", "--header", "badheader", "--number", "1"})
	if err == nil || !strings.Contains(err.Error(), "header format must be 'key=value'") {
		t.Errorf("expected header format error, got %v", err)
	}
}

func TestGenerateLogs_LoggerCreationFailure(t *testing.T) {
	// Use context with timeout to prevent hanging tests
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	orig := customLoggerFactory
	customLoggerFactory = func() (*zap.Logger, error) { return nil, errors.New("fail logger") }
	defer func() { customLoggerFactory = orig }()
	app := cli.NewApp()
	app.Commands = []*cli.Command{genLogsCommand()}
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "output"},
		&cli.IntFlag{Name: "number"},
	}
	err := app.RunContext(ctx, []string{"trazr-gen", "logs", "multi", "--output", "foo", "--number", "1"})
	if err == nil || !strings.Contains(err.Error(), "failed to create logger: fail logger") {
		t.Errorf("expected logger creation error, got %v", err)
	}
}

func TestNewCustomLogger(t *testing.T) {
	customLogger, err := newCustomLogger()
	if err != nil {
		t.Fatalf("failed to create custom logger: %v", err)
	}
	if customLogger == nil {
		t.Fatal("expected logger to be non-nil")
	}
}
