package compatibility

import (
	"testing"
)

func TestTagCommandCompatibility(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "tag_show_no_tags",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"list"}, // Add list to see if deployment was created
			},
			TestCmds: [][]string{{"tag", "Counter"}},
		},
		{
			Name: "tag_add_single",
			SetupCmds: [][]string{
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
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				// Tag by contract name first, then by address
				// Using predictable anvil deployment address
				{"tag", "Counter", "--add", "v1.0.0"},
				{"tag", "0x5FbDB2315678afecb367f032d93F642f64180aa3", "--network", "anvil-31337"},
			},
		},
		{
			Name: "tag_with_label",
			SetupCmds: [][]string{
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
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},
			},
			TestCmds: [][]string{
				// This should trigger interactive selection in v1,
				// but in test mode it should fail with multiple matches
				{"tag", "Counter", "--add", "v1.0.0", "--non-interactive"},
			},
			ExpectErr: true,
		},
	}

	RunCompatibilityTests(t, tests)
}
