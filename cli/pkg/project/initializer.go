package project

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Initializer handles project setup and initialization
type Initializer struct {
	projectName string
	createx     bool
}

// NewInitializer creates a new project initializer
func NewInitializer(projectName string, createx bool) *Initializer {
	return &Initializer{
		projectName: projectName,
		createx:     createx,
	}
}

// Initialize sets up fdeploy in an existing Foundry project
func (i *Initializer) Initialize() error {
	// Check if this is a Foundry project
	if err := i.validateFoundryProject(); err != nil {
		return err
	}

	steps := []func() error{
		i.createDeploymentsDir,
		i.createRegistry,
		i.updateRemappings,
		i.setupFdeployLibrary,
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

func (i *Initializer) createDeploymentsDir() error {
	if err := os.MkdirAll("deployments", 0755); err != nil {
		return fmt.Errorf("failed to create deployments directory: %w", err)
	}
	fmt.Println("📁 Created deployments directory")
	return nil
}

func (i *Initializer) updateRemappings() error {
	// Read existing remappings if they exist
	existingContent := ""
	if data, err := os.ReadFile("remappings.txt"); err == nil {
		existingContent = string(data)
	}

	// Add forge-deploy-lib remapping if not present
	fdeployRemapping := "forge-deploy-lib/=forge-deploy-lib/src/"
	if !contains(existingContent, "forge-deploy-lib/") {
		if existingContent != "" && !strings.HasSuffix(existingContent, "\n") {
			existingContent += "\n"
		}
		existingContent += fdeployRemapping + "\n"
	}

	if err := os.WriteFile("remappings.txt", []byte(existingContent), 0644); err != nil {
		return fmt.Errorf("failed to update remappings.txt: %w", err)
	}

	fmt.Println("📝 Updated remappings.txt with forge-deploy-lib")
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)+1] == substr+"\n" || s[len(s)-len(substr)-1:] == "\n"+substr || strings.Contains(s, "\n"+substr+"\n"))))
}

func (i *Initializer) setupFdeployLibrary() error {
	// Check if forge-deploy-lib is already installed
	if _, err := os.Stat("forge-deploy-lib"); err == nil {
		fmt.Println("📦 forge-deploy-lib already installed")
		return nil
	}

	fmt.Println("📦 Installing forge-deploy-lib...")
	
	// Install forge-deploy-lib
	// In real usage, this would be from a GitHub repo
	if err := i.runCommand("forge", "install", "your-org/forge-deploy-lib", "--no-deps"); err != nil {
		fmt.Println("⚠️  Could not install forge-deploy-lib automatically")
		fmt.Println("   Please install manually with: forge install your-org/forge-deploy-lib")
		return nil // Don't fail the init process
	}

	fmt.Println("✅ forge-deploy-lib installed")
	return nil
}

func (i *Initializer) createExampleEnvironment() error {
	// Only create .env.example if it doesn't exist
	if _, err := os.Stat(".env.example"); err == nil {
		fmt.Println("📝 .env.example already exists")
		return nil
	}

	envExample := `# fdeploy Configuration

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

	registry := fmt.Sprintf(`{
  "project": {
    "name": "%s",
    "version": "0.1.0",
    "commit": "",
    "timestamp": ""
  },
  "networks": {}
}`, i.projectName)

	if err := os.WriteFile("deployments.json", []byte(registry), 0644); err != nil {
		return fmt.Errorf("failed to create deployments.json: %w", err)
	}

	fmt.Println("📋 Created deployments.json registry")
	return nil
}

func (i *Initializer) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (i *Initializer) printNextSteps() {
	fmt.Println("")
	fmt.Println("🎉 fdeploy initialized successfully!")
	fmt.Println("")
	fmt.Println("📋 Next steps:")
	fmt.Println("1. Copy .env.example to .env and configure your values")
	if !i.fileExists("forge-deploy-lib") {
		fmt.Println("2. Install forge-deploy-lib:")
		fmt.Println("   forge install your-org/forge-deploy-lib")
	}
	fmt.Println("3. Create your first deployment script in script/")
	fmt.Println("   Example: script/DeployMyContract.s.sol")
	fmt.Println("4. Run: fdeploy predict MyContract --env staging")
	fmt.Println("5. Run: fdeploy deploy MyContract --env staging")
	fmt.Println("")
	fmt.Printf("📁 Project: %s\n", i.projectName)
}

func (i *Initializer) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}