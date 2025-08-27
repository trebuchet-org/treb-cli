package live

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trebuchet-org/treb-cli/test/helpers"
	i "github.com/trebuchet-org/treb-cli/test/integration"
)

func TestSafeDirectExecution(t *testing.T) {
	entropy, err := randomHex(8)
	if err != nil {
		t.Fatal(err)
	}
	label := fmt.Sprintf("test-label-%s", entropy)
	tests := []i.IntegrationTest{
		{
			SkipGolden:  true,
			Name:        "simple",
			Normalizers: []helpers.Normalizer{},
			OutputArtifacts: []string{
				".treb/deployments.json",
				".treb/transactions.json",
				".treb/registry.json",
				".treb/safe-txs.json",
			},
			SetupCmds: [][]string{
				s("config set namespace live"),
				s("config set network base-sepolia"),
				s("gen deploy src/UpgradeableCounter.sol:UpgradeableCounter --proxy --proxy-contract ERC1967Proxy"),
			},
			PostSetup: func(t *testing.T, ctx *helpers.TestContext) {
				if err := setDeployer(ctx.WorkDir, "DeployUpgradeableCounterProxy.s.sol", "safe0"); err != nil {
					t.Fatal(err)
				}
			},
			TestCmds: [][]string{
				sf("run DeployUpgradeableCounterProxy -e implementationLabel=%s -e proxyLabel=%s", label, label),
			},
			Test: func(t *testing.T, ctx *helpers.TestContext, output *helpers.TestOutput) {
				stdout := output.Get("stdout")
				assert.Contains(t, stdout, "ContractCreation(newContract: UpgradeableCounter")
				assert.Contains(t, stdout, "ContractCreation(newContract: ERC1967Proxy")
				assert.Contains(t, stdout, fmt.Sprintf("UpgradeableCounter:%s", label))
				assert.Contains(t, stdout, fmt.Sprintf("ERC1967Proxy:%s", label))

				if deploymentIds, err := output.DeploymentIds(); err != nil {
					t.Fatal(err)
				} else {
					assert.Len(t, deploymentIds, 2)
				}

				if txIds, err := output.TransactionIds(); err != nil {
					t.Fatal(err)
				} else {
					assert.Len(t, txIds, 1)
				}

				if safeTxHashes, err := output.SafeTxHashes(); err != nil {
					t.Fatal(err)
				} else {
					assert.Len(t, safeTxHashes, 1)
				}
			},
		},
	}

	i.RunIntegrationTests(t, tests)
}
