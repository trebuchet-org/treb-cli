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
		t.Run(tt.name, func(t *testing.T) {
			output, err := runTreb(t, tt.args...)

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
	// Test that ambiguous contract names fail in non-interactive mode
	output, err := runTreb(t, "gen", "deploy", "Counter", "--strategy", "CREATE3", "--non-interactive")
	assert.Error(t, err)
	assert.Contains(t, output, "multiple contracts found matching")

	// Test that help shows non-interactive flag
	output, err = runTreb(t, "--help")
	assert.NoError(t, err)
	assert.Contains(t, output, "non-interactive")
}

// Test command structure
func TestCommandStructure(t *testing.T) {
	commands := []string{"run", "gen", "show", "verify", "list", "init", "version", "sync", "tag"}

	for _, cmd := range commands {
		t.Run(fmt.Sprintf("%s command exists", cmd), func(t *testing.T) {
			// Some commands may error without args, but --help should work
			// or they should at least be recognized commands
			output, _ := runTreb(t, cmd, "--help")
			assert.NotContains(t, output, "unknown command")
		})
	}

	// Test gen subcommands
	t.Run("gen has deploy subcommand", func(t *testing.T) {
		output, _ := runTreb(t, "gen", "--help")
		assert.Contains(t, output, "deploy")
	})

	t.Run("gen has library subcommand", func(t *testing.T) {
		output, _ := runTreb(t, "gen", "--help")
		assert.Contains(t, output, "library")
	})
}
