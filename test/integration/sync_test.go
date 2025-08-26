package integration

import (
	"testing"
)

func TestSyncCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "sync_empty_registry",
			TestCmds: [][]string{
				{"sync"},
			},
		},
		{
			Name: "sync_with_deployments",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"sync"},
			},
		},
	}

	RunIntegrationTests(t, tests)
}
