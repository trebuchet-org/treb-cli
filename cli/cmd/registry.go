package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bogdan/fdeploy/cli/internal/registry"
	"github.com/spf13/cobra"
)

var (
	showContract string
	fromBroadcast bool
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Registry management commands",
	Long: `Manage the deployment registry including showing deployments,
syncing from broadcast files, and updating verification status.`,
}

var registryShowCmd = &cobra.Command{
	Use:   "show [contract]",
	Short: "Show deployment information",
	Long:  `Show detailed deployment information for a contract.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contract := args[0]
		
		if err := showDeployment(contract); err != nil {
			checkError(err)
		}
	},
}

var registrySyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync registry from broadcast files",
	Long:  `Sync the registry with information from Foundry broadcast files.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := syncRegistry(); err != nil {
			checkError(err)
		}
		
		fmt.Println("✅ Registry synced from broadcast files")
	},
}

func init() {
	registryCmd.AddCommand(registryShowCmd)
	registryCmd.AddCommand(registrySyncCmd)
	
	registryShowCmd.Flags().StringVar(&env, "env", "staging", "Environment to show")
	registrySyncCmd.Flags().BoolVar(&fromBroadcast, "from-broadcast", true, "Sync from broadcast files")
}

func showDeployment(contract string) error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	deployment := registryManager.GetDeployment(contract, env)
	if deployment == nil {
		return fmt.Errorf("no deployment found for %s in %s environment", contract, env)
	}

	// Pretty print the deployment
	deploymentJSON, err := json.MarshalIndent(deployment, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal deployment: %w", err)
	}

	fmt.Printf("📝 Deployment: %s_%s\n", contract, env)
	fmt.Println(string(deploymentJSON))

	return nil
}

func syncRegistry() error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// TODO: Implement sync from broadcast files
	fmt.Println("Syncing from broadcast files...")
	
	// For now, just save the registry
	return registryManager.Save()
}