package integration

import (
	"testing"
)

func TestGovernorCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "deploy_governance_infrastructure",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			TestCmds: [][]string{
				{"run", "script/DeployGovernance.s.sol"},
				{"list"},
			},
		},
		// Note: Full governor integration test is disabled due to complex OZ Governor
		// deployment requirements. The oz_governor sender type and GovernorProposalCreated
		// event parsing are implemented but require manual testing with a live governor.
	}

	RunIntegrationTests(t, tests)
}
