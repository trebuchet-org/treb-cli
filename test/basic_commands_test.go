package integration_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Table-driven tests for basic commands
func TestBasicCommands(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		contains string
	}{
		{
			name:     "version",
			args:     []string{"version"},
			contains: "treb",
		},
		{
			name:     "help",
			args:     []string{"--help"},
			contains: "non-interactive",
		},
		{
			name:     "help mentions run",
			args:     []string{"--help"},
			contains: "run",
		},
		{
			name: "list command",
			args: []string{"list"},
		},
		{
			name:     "run help",
			args:     []string{"run", "--help"},
			contains: "Run",
		},
		{
			name:     "gen help",
			args:     []string{"gen", "--help"},
			contains: "Generate",
		},
		{
			name:     "show help",
			args:     []string{"show", "--help"},
			contains: "Show",
		},
		{
			name:     "verify help",
			args:     []string{"verify", "--help"},
			contains: "Verify",
		},
		{
			name:    "invalid command",
			args:    []string{"invalid-command"},
			wantErr: true,
		},
		{
			name:    "show without args",
			args:    []string{"show"},
			wantErr: true,
		},
		{
			name:    "verify without args",
			args:    []string{"verify"},
			wantErr: true,
		},
		{
			name:     "invalid flag",
			args:     []string{"--invalid-flag"},
			wantErr:  true,
			contains: "unknown flag",
		},
	}

	for _, tt := range tests {
		tt := tt // capture loop variable
		IsolatedTest(t, tt.name, func(t *testing.T, ctx *TrebContext) {
			output, err := ctx.treb(tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.contains != "" {
				assert.Contains(t, output, tt.contains)
			}
		})
	}
}

// Test non-interactive mode
func TestNonInteractiveMode(t *testing.T) {
	IsolatedTest(t, "non_interactive_mode", func(t *testing.T, ctx *TrebContext) {
		// Test that ambiguous contract names fail in non-interactive mode
		output, err := ctx.treb("gen", "deploy", "Counter", "--strategy", "CREATE3")
		assert.Error(t, err)
		assert.Contains(t, output, "multiple contracts found matching")

		// Test that help shows non-interactive flag
		output, err = ctx.treb("--help")
		assert.NoError(t, err)
		assert.Contains(t, output, "non-interactive")
	})
}

// Test command structure
func TestCommandStructure(t *testing.T) {
	commands := []string{"run", "gen", "show", "verify", "list", "init", "version", "sync", "tag"}

	for _, cmd := range commands {
		cmd := cmd // capture loop variable
		IsolatedTest(t, fmt.Sprintf("%s_command_exists", cmd), func(t *testing.T, ctx *TrebContext) {
			// Some commands may error without args, but --help should work
			// or they should at least be recognized commands
			output, _ := ctx.treb(cmd, "--help")
			assert.NotContains(t, output, "unknown command")
		})
	}

	// Test gen subcommands
	IsolatedTest(t, "gen_has_deploy_subcommand", func(t *testing.T, ctx *TrebContext) {
		output, _ := ctx.treb("gen", "--help")
		assert.Contains(t, output, "deploy")
	})
}
