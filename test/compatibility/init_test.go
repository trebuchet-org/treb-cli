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
				trebDir := filepath.Join(helpers.GetFixtureDir(), ".treb")
				os.RemoveAll(trebDir)
			},
			TestCmds: [][]string{
				{"init", "test-project"},
			},
		},
		{
			Name: "init_existing_project",
			TestCmds: [][]string{
				{"init", "test-project"}, // Should handle gracefully
			},
		},
		{
			Name: "init_and_deploy",
			PreSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				// Remove existing .treb directory if present
				trebDir := filepath.Join(helpers.GetFixtureDir(), ".treb")
				os.RemoveAll(trebDir)
			},
			TestCmds: [][]string{
				{"init", "test-project"},
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"list"},
			},
		},
		{
			Name: "init_with_custom_namespace",
			PreSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				// Remove existing .treb directory if present
				trebDir := filepath.Join(helpers.GetFixtureDir(), ".treb")
				os.RemoveAll(trebDir)
			},
			TestCmds: [][]string{
				{"init", "prod-project", "--namespace", "production"},
				{"config", "show"},
			},
		},
	}

	RunCompatibilityTests(t, tests)
}