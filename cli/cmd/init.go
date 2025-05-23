package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	createxFlag bool
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new fdeploy project",
	Long: `Initialize a new fdeploy project with enhanced registry and optional CreateX integration.

This command sets up:
- Project structure with lib submodule
- forge-deploy-lib as git submodule
- Initial registry configuration
- Foundry project setup`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		
		if err := initProject(projectName); err != nil {
			checkError(err)
		}
		
		fmt.Printf("✅ Initialized fdeploy project: %s\n", projectName)
	},
}

func init() {
	initCmd.Flags().BoolVar(&createxFlag, "createx", true, "Initialize with CreateX integration")
}

func initProject(projectName string) error {
	// Create foundry.toml
	foundryConfig := fmt.Sprintf(`[profile.default]
src = "src"
out = "out"
libs = ["lib"]
script = "script"
test = "test"
cache_path = "cache"
broadcast = "broadcast"

# See more config options https://github.com/foundry-rs/foundry/tree/master/config

[profile.default.optimizer]
enabled = true
runs = 200

[profile.default.model_checker]
contracts = {}
engine = 'chc'
timeout = 10000
targets = ['assert', 'outOfBounds', 'overflow', 'underflow', 'divByZero']

[etherscan]
mainnet = { key = "${ETHERSCAN_API_KEY}" }
sepolia = { key = "${ETHERSCAN_API_KEY}" }
polygon = { key = "${POLYGONSCAN_API_KEY}" }
arbitrum = { key = "${ARBISCAN_API_KEY}" }

[rpc_endpoints]
mainnet = "${MAINNET_RPC_URL}"
sepolia = "${SEPOLIA_RPC_URL}"
polygon = "${POLYGON_RPC_URL}"
arbitrum = "${ARBITRUM_RPC_URL}"
`)

	if err := os.WriteFile("foundry.toml", []byte(foundryConfig), 0644); err != nil {
		return fmt.Errorf("failed to create foundry.toml: %w", err)
	}

	// Create .env.example
	envExample := `# Private keys (for deployment)
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

# Deployment environment
DEPLOYMENT_ENV=staging
`

	if err := os.WriteFile(".env.example", []byte(envExample), 0644); err != nil {
		return fmt.Errorf("failed to create .env.example: %w", err)
	}

	// Create .gitignore
	gitignore := `# Foundry
cache/
out/
broadcast/
.env

# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
go.work

# Registry (should be committed)
!deployments.json

# IDE
.vscode/
.idea/
*.swp
*.swo
`

	if err := os.WriteFile(".gitignore", []byte(gitignore), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Create initial registry
	registryContent := fmt.Sprintf(`{
  "project": {
    "name": "%s",
    "version": "0.1.0",
    "commit": "",
    "timestamp": ""
  },
  "networks": {}
}`, projectName)

	if err := os.WriteFile("deployments.json", []byte(registryContent), 0644); err != nil {
		return fmt.Errorf("failed to create deployments.json: %w", err)
	}

	// Create basic directories
	dirs := []string{"src", "script", "test"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create remappings.txt
	remappings := `forge-std/=lib/forge-std/src/
forge-deploy-lib/=lib/forge-deploy-lib/src/
createx/=lib/forge-deploy-lib/lib/createx-forge/src/
`

	if err := os.WriteFile("remappings.txt", []byte(remappings), 0644); err != nil {
		return fmt.Errorf("failed to create remappings.txt: %w", err)
	}

	fmt.Println("📦 Created project structure")
	fmt.Println("📝 Created foundry.toml, .env.example, .gitignore")
	fmt.Println("📋 Created initial registry: deployments.json")
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("1. Add forge-deploy-lib as submodule: git submodule add <repo-url> lib/forge-deploy-lib")
	fmt.Println("2. Copy .env.example to .env and fill in your values")
	fmt.Println("3. Install forge dependencies: forge install")

	return nil
}