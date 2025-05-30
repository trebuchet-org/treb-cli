package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate example deployment scripts",
	Long: `Generate example deployment scripts that work with treb's new architecture.

This command creates template scripts that demonstrate how to use treb-sol's
base contracts for deployments. The generated scripts are starting points
that you can customize for your specific deployment needs.`,
}

var genExampleCmd = &cobra.Command{
	Use:   "example [name]",
	Short: "Generate an example deployment script",
	Long: `Generate an example deployment script that demonstrates best practices.

Examples:
  treb gen example                    # Generate Deploy.s.sol
  treb gen example MyDeploy          # Generate MyDeploy.s.sol
  treb gen example deploy/Token      # Generate script/deploy/Token.s.sol`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scriptName := "Deploy"
		if len(args) > 0 {
			scriptName = args[0]
		}

		// Determine script path
		scriptPath := filepath.Join("script", scriptName+".s.sol")
		
		// Ensure script directory exists
		scriptDir := filepath.Dir(scriptPath)
		if err := os.MkdirAll(scriptDir, 0755); err != nil {
			return fmt.Errorf("failed to create script directory: %w", err)
		}

		// Check if script already exists
		if _, err := os.Stat(scriptPath); err == nil {
			return fmt.Errorf("script already exists: %s", scriptPath)
		}

		// Generate example script
		content := generateExampleScript(filepath.Base(scriptName))
		
		// Write script file
		if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write script: %w", err)
		}

		fmt.Printf("✅ Generated example script: %s\n\n", scriptPath)
		fmt.Println("This script demonstrates:")
		fmt.Println("  • Using treb-sol's Script base contract")
		fmt.Println("  • Accessing configured senders")
		fmt.Println("  • Deploying contracts with CREATE2/CREATE3")
		fmt.Println("  • Looking up existing deployments from registry")
		fmt.Println("  • Emitting deployment events for automatic tracking")
		fmt.Println("\nCustomize the script for your specific needs, then run with:")
		fmt.Printf("  treb run %s --network <network>\n", scriptPath)
		
		return nil
	},
}

var genLibraryCmd = &cobra.Command{
	Use:   "library [name]",
	Short: "Generate a library deployment script",
	Long: `Generate a deployment script specifically for Solidity libraries.

Examples:
  treb gen library                    # Generate DeployLibrary.s.sol
  treb gen library StringUtils       # Generate DeployStringUtils.s.sol`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		libraryName := "Library"
		if len(args) > 0 {
			libraryName = args[0]
		}

		// Script path
		scriptPath := filepath.Join("script", "deploy", fmt.Sprintf("Deploy%s.s.sol", libraryName))
		
		// Ensure script directory exists
		scriptDir := filepath.Dir(scriptPath)
		if err := os.MkdirAll(scriptDir, 0755); err != nil {
			return fmt.Errorf("failed to create script directory: %w", err)
		}

		// Check if script already exists
		if _, err := os.Stat(scriptPath); err == nil {
			return fmt.Errorf("script already exists: %s", scriptPath)
		}

		// Generate library script
		content := generateLibraryScript(libraryName)
		
		// Write script file
		if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write script: %w", err)
		}

		fmt.Printf("✅ Generated library deployment script: %s\n\n", scriptPath)
		fmt.Println("To deploy the library:")
		fmt.Printf("  1. Update the script with your library's artifact path\n")
		fmt.Printf("  2. Run: treb run %s --network <network>\n", scriptPath)
		
		return nil
	},
}

func init() {
	genCmd.AddCommand(genExampleCmd)
	genCmd.AddCommand(genLibraryCmd)
}

// generateExampleScript creates an example deployment script
func generateExampleScript(name string) string {
	return fmt.Sprintf(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script} from "treb-sol/src/Script.sol";

contract %s is Script {
    function run() public {
        // Get the default sender (configured via treb profile)
        Sender deployer = sender("default");
        
        // Example: Deploy a simple contract
        address myContract = deployer.deployCreate3("MyContract.sol:MyContract");
        
        // Example: Deploy with constructor arguments
        bytes memory args = abi.encode(1000, "Hello");
        address tokenContract = deployer.deployCreate3("Token.sol:Token", args);
        
        // Example: Deploy with specific salt for deterministic address
        bytes32 salt = keccak256("my-unique-salt");
        address deterministicContract = deployer.deployCreate3(
            salt,
            getCode("DeterministicContract.sol:DeterministicContract"),
            ""
        );
        
        // Example: Look up existing deployment from registry
        address existingContract = getDeployment("ExistingContract");
        
        // Example: Deploy proxy pattern
        address implementation = deployer.deployCreate3("MyImplementation.sol:MyImplementation");
        bytes memory proxyArgs = abi.encode(implementation, "");
        address proxy = deployer.deployCreate3("ERC1967Proxy.sol:ERC1967Proxy", proxyArgs);
        
        // All deployments are automatically tracked via events
        // The treb CLI will parse these events and update the registry
    }
    
    // Helper function to get bytecode
    function getCode(string memory what) internal returns (bytes memory) {
        return vm.getCode(what);
    }
}
`, name)
}

// generateLibraryScript creates a library deployment script
func generateLibraryScript(libraryName string) string {
	return fmt.Sprintf(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {LibraryDeployment} from "treb-sol/src/LibraryDeployment.sol";

contract Deploy%s is LibraryDeployment {
    // LibraryDeployment automatically:
    // 1. Reads LIBRARY_ARTIFACT_PATH from environment
    // 2. Deploys the library using CREATE2
    // 3. Emits deployment events for registry tracking
    
    // You can override the deployment logic if needed:
    /*
    function run() public override {
        // Custom deployment logic
        super.run(); // Call parent to maintain event emissions
    }
    */
}
`, libraryName)
}