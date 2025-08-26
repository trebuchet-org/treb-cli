package integration

import (
	"testing"
)

func TestTagCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "tag_show_no_tags",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"list"}, // Add list to see if deployment was created
			},
			TestCmds: [][]string{{"tag", "Counter"}},
		},
		{
			Name: "tag_add_single",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--add", "v1.0.0"},
				{"tag", "Counter"}, // Show tags after adding
			},
		},
		{
			Name: "tag_add_multiple",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--add", "v1.0.0"},
				{"tag", "Counter", "--add", "v1.0.1"},
				{"tag", "Counter", "--add", "stable"},
				{"tag", "Counter"}, // Show all tags
			},
		},
		{
			Name: "tag_add_duplicate",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"tag", "Counter", "--add", "v1.0.0"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--add", "v1.0.0"}, // Try to add duplicate
			},
		},
		{
			Name: "tag_remove_existing",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"tag", "Counter", "--add", "v1.0.0"},
				{"tag", "Counter", "--add", "v1.0.1"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--remove", "v1.0.0"},
				{"tag", "Counter"}, // Show remaining tags
			},
		},
		{
			Name: "tag_remove_non_existing",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--remove", "v1.0.0"}, // Try to remove non-existing tag
			},
		},
		{
			Name: "tag_by_address",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				// Tag by contract name first, then by address
				// Using predictable anvil deployment address
				{"tag", "Counter", "--add", "v1.0.0"},
				{"tag", "0x74148047D6bDf624C94eFc07F60cEE7b6052FB29", "--network", "anvil-31337"},
			},
		},
		{
			Name: "tag_with_label",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=test"},
			},
			TestCmds: [][]string{
				{"tag", "Counter:test", "--add", "v1.0.0"},
				{"tag", "Counter:test"},
			},
		},
		{
			Name:      "tag_non_existing_deployment",
			TestCmds:  [][]string{{"tag", "NonExisting"}},
			ExpectErr: true,
		},
		{
			Name: "tag_add_and_remove_both",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--add", "v1.0.0", "--remove", "v1.0.1"},
			},
			ExpectErr: true,
		},
		{
			Name: "tag_with_namespace",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--add", "prod-v1", "--namespace", "production"},
				{"tag", "Counter", "--namespace", "production"},
			},
		},
		{
			Name: "tag_with_multiple_deployments_same_name",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
			},
			TestCmds: [][]string{
				// This should trigger interactive selection in v1,
				// but in test mode it should fail with multiple matches
				{"tag", "Counter", "--add", "v1.0.0", "--non-interactive"},
			},
			ExpectErr: true,
		},
		// Additional test cases for more comprehensive coverage
		{
			Name: "tag_multiple_contracts_different_tags",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--add", "v1.0.0"},
				{"tag", "SampleToken", "--add", "token-v1"},
				{"list"}, // Verify both contracts exist with tags
			},
		},
		{
			Name: "tag_with_special_characters",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--add", "release/2024.01"},
				{"tag", "Counter", "--add", "hotfix-123"},
				{"tag", "Counter", "--add", "RC_1.0.0"},
				{"tag", "Counter"},
			},
		},
		{
			Name: "tag_deployment_by_id",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				// Tag using full deployment ID format
				{"tag", "default/31337/Counter", "--add", "v1.0.0"},
				{"show", "Counter"},
			},
		},
		{
			Name: "tag_cross_namespace",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "staging"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--add", "default-v1"},
				{"tag", "Counter", "--add", "staging-v1", "--namespace", "staging"},
				{"list"}, // Show both deployments with their tags
				{"list", "--namespace", "staging"},
			},
		},
		{
			Name: "tag_library_deployment",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/StringUtils.sol:StringUtils"},
				{"run", "script/deploy/DeployStringUtils.s.sol"},
			},
			TestCmds: [][]string{
				{"tag", "StringUtils", "--add", "lib-v1.0.0"},
				{"show", "StringUtils"},
			},
		},
		{
			Name: "tag_with_network_and_namespace",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production", "--network", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production", "--network", "anvil-31338"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--add", "v1.0.0", "--namespace", "production", "--network", "anvil-31337"},
				{"show", "Counter", "--namespace", "production", "--network", "anvil-31337"},
			},
		},
		{
			Name: "tag_long_tag_name",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--add", "this-is-a-very-long-tag-name-that-should-still-work-v1.0.0-beta.1"},
				{"tag", "Counter"},
			},
		},
		{
			Name: "tag_remove_all_tags",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"tag", "Counter", "--add", "v1.0.0"},
				{"tag", "Counter", "--add", "v1.0.1"},
				{"tag", "Counter", "--add", "latest"},
			},
			TestCmds: [][]string{
				{"tag", "Counter", "--remove", "v1.0.0"},
				{"tag", "Counter", "--remove", "v1.0.1"},
				{"tag", "Counter", "--remove", "latest"},
				{"tag", "Counter"}, // Should show no tags
			},
		},
		{
			Name: "tag_after_deployment_update",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v1"},
				{"tag", "Counter:v1", "--add", "initial"},
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v2"},
			},
			TestCmds: [][]string{
				{"tag", "Counter:v2", "--add", "updated"},
				{"show", "Counter:v1"}, // v1 should still have its tag
				{"show", "Counter:v2"}, // v2 should have new tag
			},
		},
	}

	RunIntegrationTests(t, tests)
}
