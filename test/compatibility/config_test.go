package compatibility

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestConfigCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "config show when no config exists",
			TestCmds: [][]string{
				{"config"},
			},
		},
		{
			Name: "config set namespace",
			TestCmds: [][]string{
				{"config", "set", "namespace", "production"},
				{"config"}, // Show to verify
			},
		},
		{
			Name: "config set namespace with ns alias",
			TestCmds: [][]string{
				{"config", "set", "ns", "testing"},
				{"config"}, // Show to verify
			},
		},
		{
			Name: "config set network",
			TestCmds: [][]string{
				{"config", "set", "network", "celo"},
				{"config"}, // Show to verify
			},
		},
		{
			Name: "config set both namespace and network",
			TestCmds: [][]string{
				{"config", "set", "namespace", "staging"},
				{"config", "set", "network", "polygon"},
				{"config"}, // Show to verify
			},
		},
		{
			Name: "config set invalid key",
			TestCmds: [][]string{
				{"config", "set", "invalid", "value"},
			},
			ExpectErr: ErrorBoth,
		},
		{
			Name: "config remove namespace",
			PreSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				// Create config with namespace set
				configPath := filepath.Join(ctx.GetWorkDir(), ".treb", "config.local.json")
				os.MkdirAll(filepath.Dir(configPath), 0755)
				os.WriteFile(configPath, []byte(`{"namespace": "production", "network": ""}`), 0644)
			},
			TestCmds: [][]string{
				{"config", "remove", "namespace"},
				{"config"}, // Show to verify
			},
		},
		{
			Name: "config remove network",
			PreSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				// Create config with network set
				// Use absolute path based on the test's working directory
				configPath := filepath.Join(ctx.GetWorkDir(), ".treb", "config.local.json")
				cwd, _ := os.Getwd()
				if os.Getenv("TREB_TEST_DEBUG") != "" {
					t.Logf("PreSetup: Creating config file at %s (cwd: %s, workDir: %s)", configPath, cwd, ctx.GetWorkDir())
				}
				os.MkdirAll(filepath.Dir(configPath), 0755)
				err := os.WriteFile(configPath, []byte(`{"namespace": "default", "network": "celo"}`), 0644)
				if os.Getenv("TREB_TEST_DEBUG") != "" {
					if err != nil {
						t.Logf("PreSetup: Failed to write config file: %v", err)
					} else {
						t.Logf("PreSetup: Config file created successfully")
						// Verify it exists
						if _, err := os.Stat(configPath); err == nil {
							t.Logf("PreSetup: Verified config file exists at %s", configPath)
						} else {
							t.Logf("PreSetup: Config file verification failed: %v", err)
						}
					}
				}
			},
			TestCmds: [][]string{
				{"config", "remove", "network"},
				{"config"}, // Show to verify
			},
		},
		{
			Name: "config remove with ns alias",
			PreSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				// Create config with namespace set
				configPath := filepath.Join(ctx.GetWorkDir(), ".treb", "config.local.json")
				os.MkdirAll(filepath.Dir(configPath), 0755)
				os.WriteFile(configPath, []byte(`{"namespace": "production", "network": ""}`), 0644)
			},
			TestCmds: [][]string{
				{"config", "remove", "ns"},
				{"config"}, // Show to verify
			},
		},
		{
			Name: "config remove invalid key",
			PreSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				// Create config file
				configPath := filepath.Join(ctx.GetWorkDir(), ".treb", "config.local.json")
				os.MkdirAll(filepath.Dir(configPath), 0755)
				os.WriteFile(configPath, []byte(`{"namespace": "default", "network": ""}`), 0644)
			},
			TestCmds: [][]string{
				{"config", "remove", "invalid"},
			},
			ExpectErr: ErrorBoth,
		},
		{
			Name: "config remove when no config exists",
			TestCmds: [][]string{
				{"config", "remove", "namespace"},
			},
			ExpectErr: ErrorBoth,
		},
		{
			Name: "config show with existing config",
			PreSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				// Create config with both values set
				configPath := filepath.Join(ctx.GetWorkDir(), ".treb", "config.local.json")
				os.MkdirAll(filepath.Dir(configPath), 0755)
				os.WriteFile(configPath, []byte(`{"namespace": "staging", "network": "polygon"}`), 0644)
			},
			TestCmds: [][]string{
				{"config"},
			},
		},
		{
			Name: "config show with network not set",
			PreSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				// Create config with empty network
				configPath := filepath.Join(ctx.GetWorkDir(), ".treb", "config.local.json")
				os.MkdirAll(filepath.Dir(configPath), 0755)
				os.WriteFile(configPath, []byte(`{"namespace": "default", "network": ""}`), 0644)
			},
			TestCmds: [][]string{
				{"config"},
			},
		},
		{
			Name: "config update existing value",
			PreSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				// Create config with initial values
				configPath := filepath.Join(ctx.GetWorkDir(), ".treb", "config.local.json")
				os.MkdirAll(filepath.Dir(configPath), 0755)
				os.WriteFile(configPath, []byte(`{"namespace": "default", "network": "celo"}`), 0644)
			},
			TestCmds: [][]string{
				{"config", "set", "namespace", "production"},
				{"config", "set", "network", "polygon"},
				{"config"}, // Show to verify
			},
		},
	}

	RunCompatibilityTests(t, tests)
}
