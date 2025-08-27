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
			Name:        "direct_execution",
			Normalizers: []helpers.Normalizer{},
			SetupCmds: [][]string{
				s("config set namespace live"),
				s("config set network base-sepolia"),
				s("gen deploy src/UpgradeableCounter.sol:UpgradeableCounter --proxy --proxy-contract ERC1967Proxy"),
			},
			PostSetup: func(t *testing.T, ctx *helpers.TestContext) {
				if err := setDeployer(ctx.WorkDir, "DeployUpgradeableCounterProxy.s.sol", "anvil", "safe0"); err != nil {
					t.Fatal(err)
				}
			},
			TestCmds: [][]string{
				sf("run DeployUpgradeableCounterProxy -e implementationLabel=%s -e proxyLabel=%s", label, label),
			},
			PostTest: func(t *testing.T, ctx *helpers.TestContext, stdout string) {
				assert.Contains(t, stdout, "ContractCreation(newContract: UpgradeableCounter")
				assert.Contains(t, stdout, "ContractCreation(newContract: ERC1967Proxy")
				assert.Contains(t, stdout, fmt.Sprintf("UpgradeableCounter:%s", label))
				assert.Contains(t, stdout, fmt.Sprintf("ERC1967Proxy:%s", label))

				treb, err := helpers.NewTrebParser(ctx)
				if err != nil {
					t.Fatal(err)
				}

				assert.Len(t, treb.Deployments, 2)
				assert.Len(t, treb.Transactions, 1)
				assert.Len(t, treb.SafeTransactions, 1)

				uc, err := treb.Deployment("UpgradeableCounter")
				if err != nil {
					t.Fatal(err)
				}

				ucTxId := uc.TransactionID
				assert.Equal(t, uc.Label, label)
				assert.Equal(t, uc.Type, "SINGLETON")
				assert.Equal(t, treb.Transactions[ucTxId].Status, "EXECUTED")

				proxy, err := treb.Deployment("ERC1967Proxy")
				if err != nil {
					t.Fatal(err)
				}
				proxyTxId := proxy.TransactionID
				assert.Equal(t, proxy.Label, label)
				assert.Equal(t, proxy.Type, "PROXY")
				assert.Equal(t, ucTxId, proxyTxId)
			},
		},
	}

	i.RunIntegrationTests(t, tests)
}
