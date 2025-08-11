package integration_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/app"
	"github.com/trebuchet-org/treb-cli/internal/cli"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// TestNewArchitectureComponents tests that the new architecture components work correctly
func TestNewArchitectureComponents(t *testing.T) {
	t.Run("app_initialization", func(t *testing.T) {
		// Test that the app can be initialized properly
		cfg, err := app.LoadConfig()
		require.NoError(t, err)
		cfg.ProjectRoot = fixtureDir
		
		appInstance, err := app.InitApp(*cfg, usecase.NopProgress{})
		require.NoError(t, err)
		assert.NotNil(t, appInstance)
		assert.NotNil(t, appInstance.ListDeployments)
		assert.NotNil(t, appInstance.ShowDeployment)
	})

	t.Run("list_command_empty", func(t *testing.T) {
		IsolatedTest(t, "new_arch_list_empty", func(t *testing.T, ctx *TrebContext) {
			// Clean registry to ensure empty state
			cleanTestArtifacts(t)

			// Load config
			cfg, err := app.LoadConfig()
			require.NoError(t, err)
			cfg.ProjectRoot = fixtureDir

			// Create and execute list command
			listCmd := cli.NewListCmd(cfg)
			
			var buf bytes.Buffer
			listCmd.SetOut(&buf)
			listCmd.SetErr(&buf)
			
			err = listCmd.Execute()
			require.NoError(t, err)
			
			output := buf.String()
			assert.Contains(t, output, "No deployments found")
		})
	})

	t.Run("list_command_with_deployments", func(t *testing.T) {
		IsolatedTest(t, "new_arch_list_deployments", func(t *testing.T, ctx *TrebContext) {
			// Deploy a contract
			output, err := ctx.treb("run", "script/DeployWithTreb.s.sol",
				"--env", "deployer=anvil",
				"--env", "COUNTER_LABEL=arch-test",
				"--env", "TOKEN_LABEL=arch-test")
			require.NoError(t, err)
			assert.Contains(t, output, "Deployment Summary")

			// Load config
			cfg, err := app.LoadConfig()
			require.NoError(t, err)
			cfg.ProjectRoot = fixtureDir

			// Create and execute list command
			listCmd := cli.NewListCmd(cfg)
			
			var buf bytes.Buffer
			listCmd.SetOut(&buf)
			listCmd.SetErr(&buf)
			
			err = listCmd.Execute()
			require.NoError(t, err)
			
			output = buf.String()
			
			// Verify output contains our deployments
			assert.Contains(t, output, "Counter:arch-test")
			assert.Contains(t, output, "SampleToken:arch-test") 
			assert.Contains(t, output, "Namespace: default")
			assert.Contains(t, output, "Chain: 31337")
			assert.Contains(t, output, "Total Deployments:")
		})
	})

	t.Run("list_command_with_filters", func(t *testing.T) {
		IsolatedTest(t, "new_arch_list_filters", func(t *testing.T, ctx *TrebContext) {
			// Deploy contracts
			ctx.treb("run", "script/DeployWithTreb.s.sol",
				"--env", "deployer=anvil",
				"--env", "COUNTER_LABEL=filter-test",
				"--env", "TOKEN_LABEL=filter-test")

			// Load config
			cfg, err := app.LoadConfig()
			require.NoError(t, err)
			cfg.ProjectRoot = fixtureDir

			// Test contract filter
			listCmd := cli.NewListCmd(cfg)
			listCmd.SetArgs([]string{"--contract", "Counter"})
			
			var buf bytes.Buffer
			listCmd.SetOut(&buf)
			listCmd.SetErr(&buf)
			
			err = listCmd.Execute()
			require.NoError(t, err)
			
			output := buf.String()
			assert.Contains(t, output, "Counter:filter-test")
			assert.NotContains(t, output, "SampleToken")
		})
	})

	t.Run("use_case_direct", func(t *testing.T) {
		IsolatedTest(t, "use_case_direct", func(t *testing.T, ctx *TrebContext) {
			// Deploy a contract
			ctx.treb("run", "script/DeployWithTreb.s.sol",
				"--env", "deployer=anvil",
				"--env", "COUNTER_LABEL=usecase-test",
				"--env", "TOKEN_LABEL=usecase-test")

			// Create app instance
			cfg, err := app.LoadConfig()
			require.NoError(t, err)
			cfg.ProjectRoot = fixtureDir
			
			appInstance, err := app.InitApp(*cfg, usecase.NopProgress{})
			require.NoError(t, err)

			// Test ListDeployments use case
			result, err := appInstance.ListDeployments.Run(context.Background(), usecase.ListDeploymentsParams{})
			require.NoError(t, err)
			
			assert.Greater(t, result.Summary.Total, 0)
			assert.NotEmpty(t, result.Deployments)
			
			// Find Counter deployment
			var counterDeployment *domain.Deployment
			for _, dep := range result.Deployments {
				if dep.ContractName == "Counter" && dep.Label == "usecase-test" {
					counterDeployment = dep
					break
				}
			}
			require.NotNil(t, counterDeployment)

			// Test ShowDeployment use case
			deployment, err := appInstance.ShowDeployment.Run(context.Background(), usecase.ShowDeploymentParams{
				ID: counterDeployment.ID,
			})
			require.NoError(t, err)
			assert.Equal(t, counterDeployment.ID, deployment.ID)
			assert.Equal(t, counterDeployment.Address, deployment.Address)
		})
	})
}