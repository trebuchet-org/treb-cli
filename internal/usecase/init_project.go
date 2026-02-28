package usecase

import (
	"context"
	"fmt"
	"os"
)

// InitProject handles project initialization
type InitProject struct {
	fileWriter FileWriter
	progress   ProgressSink
}

// NewInitProject creates a new init project use case
func NewInitProject(fileWriter FileWriter, progress ProgressSink) *InitProject {
	return &InitProject{
		fileWriter: fileWriter,
		progress:   progress,
	}
}

// InitProjectResult contains the result of project initialization
type InitProjectResult struct {
	FoundryProjectValid bool
	TrebSolInstalled    bool
	RegistryCreated     bool
	TrebTomlCreated     bool
	EnvExampleCreated   bool
	AlreadyInitialized  bool
	Steps               []InitStep
}

// InitStep represents a step in the initialization process
type InitStep struct {
	Name    string
	Success bool
	Message string
	Error   error
}

// Execute initializes treb in a Foundry project
func (i *InitProject) Execute(ctx context.Context) (*InitProjectResult, error) {
	result := &InitProjectResult{
		Steps: []InitStep{},
	}

	// Check if this is a Foundry project
	if err := i.validateFoundryProject(); err != nil {
		result.Steps = append(result.Steps, InitStep{
			Name:    "Validate Foundry Project",
			Success: false,
			Error:   err,
		})
		return result, err
	}
	result.FoundryProjectValid = true
	result.Steps = append(result.Steps, InitStep{
		Name:    "Validate Foundry Project",
		Success: true,
		Message: "Valid Foundry project detected",
	})

	// Check treb-sol library
	step := i.checkTrebSolLibrary()
	result.Steps = append(result.Steps, step)
	if step.Success {
		result.TrebSolInstalled = true
	} else {
		return result, step.Error
	}

	// Create registry
	step = i.createRegistry(ctx)
	result.Steps = append(result.Steps, step)
	if step.Success {
		result.RegistryCreated = true
		if step.Message == "Registry files already exist in .treb/" {
			result.AlreadyInitialized = true
		}
	}

	// Create treb.toml
	step = i.createTrebToml(ctx)
	result.Steps = append(result.Steps, step)
	if step.Success {
		result.TrebTomlCreated = true
	}

	// Create example environment
	step = i.createExampleEnvironment(ctx)
	result.Steps = append(result.Steps, step)
	if step.Success {
		result.EnvExampleCreated = true
	}

	return result, nil
}

func (i *InitProject) validateFoundryProject() error {
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

	return nil
}

func (i *InitProject) checkTrebSolLibrary() InitStep {
	// Check if treb-sol is installed
	if _, err := os.Stat("lib/treb-sol"); err == nil {
		return InitStep{
			Name:    "Check treb-sol Library",
			Success: true,
			Message: "treb-sol library found",
		}
	}

	return InitStep{
		Name:    "Check treb-sol Library",
		Success: false,
		Message: "treb-sol library not found",
		Error:   fmt.Errorf("treb-sol library is required but not found in lib/treb-sol. Please install with: forge install trebuchet-org/treb-sol"),
	}
}

func (i *InitProject) createRegistry(ctx context.Context) InitStep {
	// Create .treb directory
	if err := i.fileWriter.EnsureDirectory(ctx, ".treb"); err != nil {
		return InitStep{
			Name:    "Create Registry",
			Success: false,
			Error:   fmt.Errorf("failed to create .treb directory: %w", err),
		}
	}

	// Check if registry files already exist
	exists, err := i.fileWriter.FileExists(ctx, ".treb/deployments.json")
	if err != nil {
		return InitStep{
			Name:    "Create Registry",
			Success: false,
			Error:   fmt.Errorf("failed to check registry: %w", err),
		}
	}

	if exists {
		return InitStep{
			Name:    "Create Registry",
			Success: true,
			Message: "Registry files already exist in .treb/",
		}
	}

	// Create empty registry files
	registryFiles := map[string]string{
		".treb/deployments.json":  "{}",
		".treb/transactions.json": "{}",
		".treb/safe-txs.json":     "{}",
		".treb/registry.json":     "{}",
	}

	for filename, content := range registryFiles {
		if err := i.fileWriter.WriteScript(ctx, filename, content); err != nil {
			return InitStep{
				Name:    "Create Registry",
				Success: false,
				Error:   fmt.Errorf("failed to create %s: %w", filename, err),
			}
		}
	}

	return InitStep{
		Name:    "Create Registry",
		Success: true,
		Message: "Created v2 registry structure in .treb/",
	}
}

func (i *InitProject) createTrebToml(ctx context.Context) InitStep {
	// Only create treb.toml if it doesn't exist
	exists, err := i.fileWriter.FileExists(ctx, "treb.toml")
	if err != nil {
		return InitStep{
			Name:    "Create treb.toml",
			Success: false,
			Error:   fmt.Errorf("failed to check treb.toml: %w", err),
		}
	}

	if exists {
		return InitStep{
			Name:    "Create treb.toml",
			Success: true,
			Message: "treb.toml already exists",
		}
	}

	trebToml := `# treb.toml — Treb configuration
#
# Accounts define signing entities (wallets, hardware wallets, multisigs).
# Namespaces map roles to accounts for different environments.
# Namespaces support dot-separated names for hierarchical inheritance
# (e.g., "production.ntt" inherits from "production", which inherits from "default").

# --- Accounts ---
# Each [accounts.<name>] defines a named signing entity.

[accounts.deployer]
type = "private_key"
private_key = "${DEPLOYER_PRIVATE_KEY}"

# --- Namespaces ---
# Each [namespace.<name>] maps roles to account names.
# The optional 'profile' key maps to a foundry.toml profile (defaults to namespace name).

[namespace.default]
profile = "default"
deployer = "deployer"

# --- Fork Configuration ---
# Uncomment to configure fork setup script for treb dev commands.
# [fork]
# setup = "script/ForkSetup.s.sol"
`

	if err := i.fileWriter.WriteScript(ctx, "treb.toml", trebToml); err != nil {
		return InitStep{
			Name:    "Create treb.toml",
			Success: false,
			Error:   fmt.Errorf("failed to create treb.toml: %w", err),
		}
	}

	return InitStep{
		Name:    "Create treb.toml",
		Success: true,
		Message: "Created treb.toml with default sender config",
	}
}

func (i *InitProject) createExampleEnvironment(ctx context.Context) InitStep {
	// Only create .env.example if it doesn't exist
	exists, err := i.fileWriter.FileExists(ctx, ".env.example")
	if err != nil {
		return InitStep{
			Name:    "Create Environment Example",
			Success: false,
			Error:   fmt.Errorf("failed to check .env.example: %w", err),
		}
	}

	if exists {
		return InitStep{
			Name:    "Create Environment Example",
			Success: true,
			Message: ".env.example already exists",
		}
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

	if err := i.fileWriter.WriteScript(ctx, ".env.example", envExample); err != nil {
		return InitStep{
			Name:    "Create Environment Example",
			Success: false,
			Error:   fmt.Errorf("failed to create .env.example: %w", err),
		}
	}

	return InitStep{
		Name:    "Create Environment Example",
		Success: true,
		Message: "Created .env.example",
	}
}
