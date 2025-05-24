package project

import (
	"fmt"
	"os"
)

// Initializer handles project setup and initialization
type Initializer struct {}

// NewInitializer creates a new project initializer
func NewInitializer() *Initializer {
	return &Initializer{}
}

// Initialize sets up treb in an existing Foundry project
func (i *Initializer) Initialize() error {
	// Check if this is a Foundry project
	if err := i.validateFoundryProject(); err != nil {
		return err
	}

	steps := []func() error{
		i.checkTrebSolLibrary,
		i.createRegistry,
		i.createExampleEnvironment,
	}

	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}

	i.printNextSteps()
	return nil
}

func (i *Initializer) validateFoundryProject() error {
	// Check for foundry.toml
	if _, err := os.Stat("foundry.toml"); os.IsNotExist(err) {
		return fmt.Errorf("foundry.toml not found - please initialize a Foundry project first with 'forge init'")
	}

	// Check for expected directories
	expectedDirs := []string{"src", "script"}
	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fmt.Errorf("directory '%s' not found - this doesn't appear to be a valid Foundry project", dir)
		}
	}

	fmt.Println("✅ Valid Foundry project detected")
	return nil
}



func (i *Initializer) checkTrebSolLibrary() error {
	// Check if treb-sol is installed
	if _, err := os.Stat("lib/treb-sol"); err == nil {
		fmt.Println("✅ treb-sol library found")
		return nil
	}

	fmt.Println("❌ treb-sol library not found")
	fmt.Println("   Please install with: forge install trebuchet-org/treb-sol")
	return fmt.Errorf("treb-sol library is required but not found in lib/treb-sol")
}

func (i *Initializer) createExampleEnvironment() error {
	// Only create .env.example if it doesn't exist
	if _, err := os.Stat(".env.example"); err == nil {
		fmt.Println("📝 .env.example already exists")
		return nil
	}

	envExample := `# treb Configuration

# Private keys (for deployment)
DEPLOYER_PRIVATE_KEY=

# RPC URLs  
MAINNET_RPC_URL=
SEPOLIA_RPC_URL=
POLYGON_RPC_URL=
ARBITRUM_RPC_URL=

# API Keys for verification
ETHERSCAN_API_KEY=
POLYGONSCAN_API_KEY=
ARBISCAN_API_KEY=

# Deployment configuration
DEPLOYMENT_ENV=staging
CONTRACT_VERSION=v0.1.0
`

	if err := os.WriteFile(".env.example", []byte(envExample), 0644); err != nil {
		return fmt.Errorf("failed to create .env.example: %w", err)
	}

	fmt.Println("📝 Created .env.example")
	return nil
}


func (i *Initializer) createRegistry() error {
	// Only create if it doesn't exist
	if _, err := os.Stat("deployments.json"); err == nil {
		fmt.Println("📋 deployments.json already exists")
		return nil
	}

	registry := `{}`

	if err := os.WriteFile("deployments.json", []byte(registry), 0644); err != nil {
		return fmt.Errorf("failed to create deployments.json: %w", err)
	}

	fmt.Println("📋 Created deployments.json registry")
	return nil
}


func (i *Initializer) printNextSteps() {
	fmt.Println("")
	fmt.Println("🎉 treb initialized successfully!")
	fmt.Println("")
	fmt.Println("📋 Next steps:")
	fmt.Println("1. Copy .env.example to .env and configure your deployment keys:")
	fmt.Println("   • Set DEPLOYER_PRIVATE_KEY for your deployment wallet")
	fmt.Println("   • Set RPC URLs for networks you'll deploy to")
	fmt.Println("   • Set API keys for contract verification")
	fmt.Println("")
	fmt.Println("2. Configure deployment environments in foundry.toml:")
	fmt.Println("   • Add [profile.staging.deployer] and [profile.production.deployer] sections")
	fmt.Println("   • See documentation for Safe multisig and hardware wallet support")
	fmt.Println("")
	fmt.Println("3. Generate your first deployment script:")
	fmt.Println("   treb generate deploy Counter")
	fmt.Println("")
	fmt.Println("4. Predict and deploy:")
	fmt.Println("   treb deploy predict Counter --network sepolia")
	fmt.Println("   treb deploy Counter --network sepolia")
	fmt.Println("")
	fmt.Println("5. View and manage deployments:")
	fmt.Println("   treb list")
	fmt.Println("   treb show Counter")
	fmt.Println("   treb tag Counter v1.0.0")
}

func (i *Initializer) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}