package compatibility

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestInitCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "init_new_project",
			PreSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				// Remove existing .treb directory if present
				trebDir := filepath.Join(ctx.GetWorkDir(), ".treb")
				os.RemoveAll(trebDir)
			},
			TestCmds: [][]string{
				{"init"},
			},
			ExpectDiff: true, // v1 and v2 have different emojis
		},
		{
			Name: "init_existing_project",
			SetupCmds: [][]string{
				{"init"}, // First init
			},
			TestCmds: [][]string{
				{"init"}, // Should handle gracefully when already initialized
			},
			ExpectDiff: true, // v1 and v2 have different messages for re-init
		},
		{
			Name: "init_and_deploy",
			PreSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				// Remove existing .treb directory if present
				trebDir := filepath.Join(ctx.GetWorkDir(), ".treb")
				os.RemoveAll(trebDir)
			},
			TestCmds: [][]string{
				{"init"},
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"list"},
			},
			ExpectDiff: true, // v1 and v2 have different output formatting
		},
	}

	RunCompatibilityTests(t, tests)
}

