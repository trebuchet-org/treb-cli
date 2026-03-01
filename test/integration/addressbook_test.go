package integration

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestAddressbookCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "addressbook_set_and_list",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"addressbook", "set", "WETH", "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"},
				{"addressbook", "set", "USDC", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"},
			},
			TestCmds: [][]string{
				{"addressbook", "list"},
			},
			OutputArtifacts: []string{".treb/addressbook.json"},
		},
		{
			Name: "addressbook_remove",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"addressbook", "set", "WETH", "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"},
				{"addressbook", "set", "USDC", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"},
				{"addressbook", "remove", "WETH"},
			},
			TestCmds: [][]string{
				{"addressbook", "list"},
			},
			OutputArtifacts: []string{".treb/addressbook.json"},
		},
		{
			Name: "addressbook_set_invalid_address",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			TestCmds: [][]string{
				{"addressbook", "set", "BAD", "not-an-address"},
			},
			ExpectErr:       true,
			OutputArtifacts: []string{},
		},
		{
			Name:            "addressbook_set_no_network",
			TestCmds:        [][]string{{"addressbook", "set", "WETH", "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"}},
			ExpectErr:       true,
			OutputArtifacts: []string{},
		},
		{
			Name: "addressbook_remove_nonexistent",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			TestCmds: [][]string{
				{"addressbook", "remove", "NOPE"},
			},
			ExpectErr:       true,
			OutputArtifacts: []string{},
		},
		{
			Name: "addressbook_default_subcommand",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"addressbook", "set", "WETH", "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"},
			},
			TestCmds: [][]string{
				{"addressbook"},
			},
			OutputArtifacts: []string{},
		},
		{
			Name: "addressbook_alias",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"ab", "set", "WETH", "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"},
			},
			TestCmds: [][]string{
				{"ab", "ls"},
			},
			OutputArtifacts: []string{},
		},
		{
			Name: "list_with_addressbook_entries",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"addressbook", "set", "WETH", "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"},
			},
			TestCmds: [][]string{
				{"list"},
			},
		},
		{
			Name: "list_with_addressbook_json",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"addressbook", "set", "WETH", "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"},
			},
			TestCmds:        [][]string{{"list", "--json"}},
			SkipGolden:      true,
			OutputArtifacts: []string{},
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				jsonStr := extractJSONObject(output)

				var result map[string]interface{}
				require.NoError(t, json.Unmarshal([]byte(jsonStr), &result))

				// Should have deployments
				entries, ok := result["deployments"].([]interface{})
				require.True(t, ok, "expected deployments array")
				require.Len(t, entries, 1)

				// Should have addressbook
				ab, ok := result["addressbook"].([]interface{})
				require.True(t, ok, "expected addressbook array")
				require.Len(t, ab, 1)
				abEntry := ab[0].(map[string]interface{})
				assert.Equal(t, "WETH", abEntry["name"])
				assert.Equal(t, "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", abEntry["address"])
			},
		},
	}

	RunIntegrationTests(t, tests)
}
