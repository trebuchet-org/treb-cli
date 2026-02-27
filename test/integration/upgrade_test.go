package integration

import (
	"testing"

	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestUpgradeCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "upgrade_custom_proxy_implementation",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				// Deploy the custom proxy with initial implementation
				{"run", "script/DeployCustomProxy.s.sol"},
			},
			TestCmds: [][]string{
				// Show state before upgrade
				{"list"},
				// Run upgrade script (deploys V2 impl + calls _setImplementation)
				{"run", "script/UpgradeCustomProxy.s.sol"},
				// Show state after upgrade - registry should reflect new implementation
				{"list"},
				{"show", "Proxy"},
			},
			Normalizers: append(
				helpers.GetDefaultNormalizers(),
				helpers.LegacySolidityNormalizer{},
			),
		},
	}

	RunIntegrationTests(t, tests)
}
