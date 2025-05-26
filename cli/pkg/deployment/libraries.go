package deployment

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// LibraryInfo represents a library and its deployment address
type LibraryInfo struct {
	Requirement contracts.LibraryRequirement
	Address     common.Address
}

// checkAndResolveLibraries checks if required libraries are deployed and prompts to deploy missing ones
func (d *DeploymentContext) checkAndResolveLibraries(contractInfo *contracts.ContractInfo) ([]LibraryInfo, error) {
	libs := contractInfo.GetRequiredLibraries()
	if len(libs) == 0 {
		return nil, nil
	}

	fmt.Printf("\n📚 Contract requires %d libraries:\n", len(libs))

	var resolvedLibs []LibraryInfo
	var missingLibs []contracts.LibraryRequirement

	// Check each library
	for _, lib := range libs {
		fmt.Printf("   - %s (%s)\n", lib.Name, lib.Path)

		// Look up library deployment on current chain (libraries don't have env)
		entry := d.findLibraryDeployment(lib.Name)
		if entry != nil {
			fmt.Printf("     ✓ Found at %s\n", entry.Address.Hex())
			resolvedLibs = append(resolvedLibs, LibraryInfo{
				Requirement: lib,
				Address:     entry.Address,
			})
		} else {
			fmt.Printf("     ✗ Not deployed on %s\n", d.networkInfo.Name)
			missingLibs = append(missingLibs, lib)
		}
	}

	// If all libraries are deployed, return
	if len(missingLibs) == 0 {
		return resolvedLibs, nil
	}

	// Ask user if they want to deploy missing libraries
	fmt.Printf("\n⚠️  Missing %d libraries on %s\n", len(missingLibs), d.networkInfo.Name)

	selector := interactive.NewSelector()
	shouldDeploy, err := selector.PromptConfirm("Would you like to deploy the missing libraries now?", true)
	if err != nil || !shouldDeploy {
		return nil, fmt.Errorf("required libraries not deployed")
	}

	// Deploy missing libraries
	fmt.Println("\n🚀 Deploying missing libraries...")
	for _, lib := range missingLibs {
		fmt.Printf("\nDeploying %s...\n", lib.Name)

		// Create a nested deployment context for the library
		libCtx, err := d.createLibraryDeploymentContext(lib.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create library deployment context: %w", err)
		}

		// Execute the library deployment
		result, err := libCtx.Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to deploy library %s: %w", lib.Name, err)
		}

		// Add to resolved libraries
		resolvedLibs = append(resolvedLibs, LibraryInfo{
			Requirement: lib,
			Address:     result.Address,
		})

		fmt.Printf("✅ Deployed %s at %s\n", lib.Name, result.Address.Hex())
	}

	return resolvedLibs, nil
}

// findLibraryDeployment looks up a library deployment on the current chain
func (d *DeploymentContext) findLibraryDeployment(libraryName string) *types.DeploymentEntry {
	chainID := fmt.Sprintf("%d", d.networkInfo.ChainID())

	// Libraries are deployed with env="default" and deploymentType=library
	for _, deployment := range d.registryManager.GetAllDeployments() {
		if deployment.ChainID == chainID &&
			deployment.Entry.ContractName == libraryName &&
			deployment.Entry.Deployment.Status == types.StatusExecuted &&
			deployment.Entry.Type == types.LibraryDeployment {
			if deployment.Entry.Address != (common.Address{}) {
				return deployment.Entry
			}
		}
	}

	return nil
}

// createLibraryDeploymentContext creates a nested deployment context for a library
func (d *DeploymentContext) createLibraryDeploymentContext(libraryName string) (*DeploymentContext, error) {
	// Create deployment params for the library
	libParams := &DeploymentParams{
		ContractQuery:  libraryName,
		NetworkName:    d.Params.NetworkName,
		Env:            "default", // Libraries always use default env
		DeploymentType: types.LibraryDeployment,
		Debug:          d.Params.Debug,
		Predict:        false, // Always execute for nested deployments
	}

	// Create new context
	libCtx := NewDeploymentContext(d.projectRoot, libParams, d.registryManager)

	// Copy network info and other state
	libCtx.networkInfo = d.networkInfo
	libCtx.forge = d.forge

	// Prepare the library deployment
	if err := libCtx.PrepareLibraryDeployment(); err != nil {
		return nil, err
	}

	// Copy base environment variables
	libCtx.envVars = make(map[string]string)
	for k, v := range d.envVars {
		libCtx.envVars[k] = v
	}

	// Override with library-specific vars
	libCtx.envVars["LIBRARY_NAME"] = libCtx.contractInfo.Name
	libCtx.envVars["LIBRARY_ARTIFACT_PATH"] = fmt.Sprintf("%s:%s", libCtx.contractInfo.Path, libCtx.contractInfo.Name)

	return libCtx, nil
}

// generateLibraryFlags generates the --libraries flags for forge script
func generateLibraryFlags(libs []LibraryInfo) []string {
	var flags []string

	for _, lib := range libs {
		// Format: path/to/Library.sol:LibraryName:0xADDRESS
		flag := fmt.Sprintf("%s:%s:%s", lib.Requirement.Path, lib.Requirement.Name, lib.Address.Hex())
		flags = append(flags, "--libraries", flag)
	}

	return flags
}
