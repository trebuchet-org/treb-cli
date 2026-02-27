package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestInitCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "init_new_project",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Remove fixture treb.toml so init creates a fresh one
				os.Remove(filepath.Join(ctx.WorkDir, "treb.toml"))
			},
			TestCmds: [][]string{
				{"init"},
			},
			OutputArtifacts: append(DefaultOutputArtifacs, "treb.toml"),
		},
		{
			Name: "init_existing_project",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Remove fixture treb.toml so first init creates a fresh one
				os.Remove(filepath.Join(ctx.WorkDir, "treb.toml"))
			},
			SetupCmds: [][]string{
				{"init"}, // First init
			},
			TestCmds: [][]string{
				{"init"}, // Should handle gracefully when already initialized
			},
			OutputArtifacts: append(DefaultOutputArtifacs, "treb.toml"),
		},
		{
			Name: "init_and_deploy",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Pre-create treb.toml with anvil sender matching the test fixture's
				// foundry.toml config so that init skips treb.toml generation and
				// subsequent run commands find the expected sender.
				trebToml := `# treb.toml — Treb sender configuration

[ns.default]
profile = "default"

[ns.default.senders.anvil]
type = "private_key"
private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
`
				err := os.WriteFile(filepath.Join(ctx.WorkDir, "treb.toml"), []byte(trebToml), 0644)
				if err != nil {
					t.Fatalf("Failed to create treb.toml: %v", err)
				}
			},
			TestCmds: [][]string{
				{"init"},
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "-n", "anvil-31337"},
				{"list"},
			},
		},
	}

	RunIntegrationTests(t, tests)
}
