package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestAutoResolveSenders(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "auto_resolve_safe_signer",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				// Deploy using auto-resolve namespace where safe0's signer (signer0)
				// is NOT explicitly mapped — should be auto-resolved from the account reference.
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "auto-resolve"},
				{"show", "Counter", "--namespace", "auto-resolve"},
			},
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				treb, err := helpers.NewTrebParser(ctx)
				if err != nil {
					t.Fatal(err)
				}
				assert.Len(t, treb.Deployments, 1)
				dep, err := treb.Deployment("Counter")
				assert.NoError(t, err)
				assert.Equal(t, "auto-resolve", dep.Namespace)
			},
		},
		{
			Name: "explicit_signer_mapping_regression",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				// Deploy using default namespace where all senders are explicitly mapped.
				// This verifies that auto-resolution doesn't break existing explicit configs.
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"show", "Counter"},
			},
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				treb, err := helpers.NewTrebParser(ctx)
				if err != nil {
					t.Fatal(err)
				}
				assert.Len(t, treb.Deployments, 1)
				dep, err := treb.Deployment("Counter")
				assert.NoError(t, err)
				assert.Equal(t, "default", dep.Namespace)
			},
		},
	}

	RunIntegrationTests(t, tests)
}
