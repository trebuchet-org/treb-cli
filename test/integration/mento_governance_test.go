package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func appendToFoundryToml(t *testing.T, ctx *helpers.TestContext, config string) {
	path := filepath.Join(ctx.WorkDir, "foundry.toml")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(config); err != nil {
		t.Fatal(err)
	}
}

func TestMentoGovernanceConfiguration(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "mento_governance_config_validation",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			TestCmds: [][]string{
				// Test that mento_governance sender type is recognized
				{"config", "show"},
			},
			OutputArtifacts: []string{".treb/config.toml"},
		},
	}

	RunIntegrationTests(t, tests)
}

func TestMentoGovernanceProposalCreation(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "governance_proposal_basic",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				s("config set namespace default"),
				{"gen", "deploy", "Counter"},
			},
			PostSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Configure a mento_governance sender in foundry.toml
				config := `
[senders]
proposer = { type = "private_key", private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" }
governance = {
    type = "mento_governance",
    governor = "0x5FbDB2315678afecb367f032d93F642f64180aa3",
    timelock = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
    proposer = "proposer",
    voting_delay = 1,
    voting_period = 50400,
    timelock_delay = 172800
}
`
				appendToFoundryToml(t, ctx, config)
			},
			TestCmds: [][]string{
				// Generate deployment script
				{"gen", "deploy", "Counter"},
				// List senders to verify governance sender is loaded
				{"config", "show"},
			},
			ExpectErr: false,
		},
	}

	RunIntegrationTests(t, tests)
}

func TestMentoGovernanceInvalidConfig(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "governance_missing_governor",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			PostSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Configure a mento_governance sender with missing governor
				config := `
[senders]
proposer = { type = "private_key", private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" }
governance = {
    type = "mento_governance",
    timelock = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
    proposer = "proposer"
}
`
				appendToFoundryToml(t, ctx, config)
			},
			TestCmds: [][]string{
				{"gen", "deploy", "Counter"},
			},
			ExpectErr: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, stdout string) {
				assert.Contains(t, stdout, "governor")
			},
		},
		{
			Name: "governance_missing_timelock",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			PostSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Configure a mento_governance sender with missing timelock
				config := `
[senders]
proposer = { type = "private_key", private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" }
governance = {
    type = "mento_governance",
    governor = "0x5FbDB2315678afecb367f032d93F642f64180aa3",
    proposer = "proposer"
}
`
				appendToFoundryToml(t, ctx, config)
			},
			TestCmds: [][]string{
				{"gen", "deploy", "Counter"},
			},
			ExpectErr: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, stdout string) {
				assert.Contains(t, stdout, "timelock")
			},
		},
		{
			Name: "governance_missing_proposer",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			PostSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Configure a mento_governance sender with missing proposer
				config := `
[senders]
governance = {
    type = "mento_governance",
    governor = "0x5FbDB2315678afecb367f032d93F642f64180aa3",
    timelock = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512"
}
`
				appendToFoundryToml(t, ctx, config)
			},
			TestCmds: [][]string{
				{"gen", "deploy", "Counter"},
			},
			ExpectErr: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, stdout string) {
				assert.Contains(t, stdout, "proposer")
			},
		},
		{
			Name: "governance_invalid_proposer_reference",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			PostSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Configure a mento_governance sender with non-existent proposer reference
				config := `
[senders]
proposer = { type = "private_key", private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" }
governance = {
    type = "mento_governance",
    governor = "0x5FbDB2315678afecb367f032d93F642f64180aa3",
    timelock = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
    proposer = "nonexistent"
}
`
				appendToFoundryToml(t, ctx, config)
			},
			TestCmds: [][]string{
				{"gen", "deploy", "Counter"},
			},
			ExpectErr: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, stdout string) {
				assert.Contains(t, stdout, "nonexistent")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

func TestMentoGovernanceTypeConstants(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "verify_sender_type_constant",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			PostSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Configure a valid mento_governance sender
				config := `
[senders]
proposer = { type = "private_key", private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" }
governance = {
    type = "mento_governance",
    governor = "0x5FbDB2315678afecb367f032d93F642f64180aa3",
    timelock = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
    proposer = "proposer",
    voting_delay = 1,
    voting_period = 50400,
    timelock_delay = 172800
}
`
				appendToFoundryToml(t, ctx, config)
			},
			TestCmds: [][]string{
				{"gen", "deploy", "Counter"},
				{"config", "show"},
			},
			ExpectErr: false,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, stdout string) {
				// Verify the governance sender appears in config
				assert.Contains(t, stdout, "governance")
				assert.Contains(t, stdout, "mento_governance")
			},
		},
	}

	RunIntegrationTests(t, tests)
}
