package cli

import (
	"testing"

	"go.uber.org/zap"
)

func TestGenDiagnosticsCommand_Structure(t *testing.T) {
	cmd := genDiagnosticsCommand()
	if cmd.Name != "diagnostics" {
		t.Errorf("expected command name 'diagnostics', got '%s'", cmd.Name)
	}
	if !cmd.Hidden {
		t.Error("expected diagnostics command to be hidden")
	}
	if len(cmd.Subcommands) != 1 {
		t.Errorf("expected 1 subcommand, got %d", len(cmd.Subcommands))
	}
	sub := cmd.Subcommands[0]
	if sub.Name != "network" {
		t.Errorf("expected subcommand 'network', got '%s'", sub.Name)
	}
	if sub.Action == nil {
		t.Error("expected subcommand to have an action")
	}
}

func TestGenDiagnosticsCommand_NetworkAction(t *testing.T) {
	origLogger := logger
	logger = zap.NewNop()
	defer func() { logger = origLogger }()

	cmd := genDiagnosticsCommand()
	if len(cmd.Subcommands) == 0 {
		t.Fatal("expected at least one subcommand")
	}
	networkCmd := cmd.Subcommands[0]
	if networkCmd.Name != "network" {
		t.Fatalf("expected subcommand 'network', got '%s'", networkCmd.Name)
	}
	if networkCmd.Action == nil {
		t.Fatal("expected network subcommand to have an action")
	}
	// Simulate a cli.Context (minimal, just for coverage)
	err := networkCmd.Action(nil)
	if err != nil {
		t.Errorf("expected nil error from network action, got %v", err)
	}
}
