package deployment

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// LibraryInfo represents a library and its deployment address
type LibraryInfo struct {
	Requirement contracts.LibraryRequirement
	Address     common.Address
}

// librarySpinner wraps the spinner functionality
type librarySpinner struct {
	s *spinner.Spinner
}

func newLibrarySpinner() *librarySpinner {
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.HideCursor = true
	return &librarySpinner{s: s}
}

func (ls *librarySpinner) Start(message string) {
	ls.s.Suffix = " " + message
	ls.s.Start()
}

func (ls *librarySpinner) Success(message string) {
	ls.s.Stop()
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Println(message)
}

func (ls *librarySpinner) Warning(message string) {
	ls.s.Stop()
	color.New(color.FgYellow).Printf("⚠ ")
	fmt.Println(message)
}

func (ls *librarySpinner) Fail(message string) {
	ls.s.Stop()
	color.New(color.FgRed).Printf("✗ ")
	fmt.Println(message)
}

// checkAndResolveLibraries checks if required libraries are deployed and prompts to deploy missing ones
func (d *DeploymentContext) checkAndResolveLibraries(contractInfo *contracts.ContractInfo) ([]LibraryInfo, error) {
	libs := contractInfo.GetRequiredLibraries()
	if len(libs) == 0 {
		return nil, nil
	}

	// Start spinner for library check
	spinner := newLibrarySpinner()
	spinner.Start(fmt.Sprintf("Checking %d required libraries", len(libs)))

	var resolvedLibs []LibraryInfo
	var missingLibs []contracts.LibraryRequirement

	// Check each library
	for _, lib := range libs {
		// Look up library deployment on current chain (libraries don't have env)
		entry := d.findLibraryDeployment(lib.Name)
		if entry != nil {
			resolvedLibs = append(resolvedLibs, LibraryInfo{
				Requirement: lib,
				Address:     entry.Address,
			})
		} else {
			missingLibs = append(missingLibs, lib)
		}
	}

	// Stop spinner with result
	if len(missingLibs) == 0 {
		spinner.Success("All libraries found")
		return resolvedLibs, nil
	}

	spinner.Warning(fmt.Sprintf("Missing %d of %d libraries on %s", len(missingLibs), len(libs), d.networkInfo.Name))

	// Ask user if they want to deploy missing libraries
	fmt.Println()
	selector := interactive.NewSelector()
	shouldDeploy, err := selector.PromptConfirm("Would you like to deploy the missing libraries now", true)
	if err != nil || !shouldDeploy {
		return nil, fmt.Errorf("required libraries not deployed")
	}

	// Deploy missing libraries
	fmt.Println()
	deploySpinner := newLibrarySpinner()

	for i, lib := range missingLibs {
		deploySpinner.Start(fmt.Sprintf("Deploying library %s (%d/%d)", lib.Name, i+1, len(missingLibs)))

		// Create a nested deployment context for the library
		libCtx, err := d.createLibraryDeploymentContext(lib.Name)
		if err != nil {
			deploySpinner.Fail(fmt.Sprintf("Failed to prepare %s deployment", lib.Name))
			return nil, fmt.Errorf("failed to create library deployment context: %w", err)
		}

		// Execute the library deployment
		result, err := libCtx.Execute()
		if err != nil {
			deploySpinner.Fail(fmt.Sprintf("Failed to deploy %s", lib.Name))
			return nil, fmt.Errorf("failed to deploy library %s: %w", lib.Name, err)
		}

		// Add to resolved libraries
		resolvedLibs = append(resolvedLibs, LibraryInfo{
			Requirement: lib,
			Address:     result.Deployed,
		})

		deploySpinner.Success(fmt.Sprintf("Deployed %s at %s", lib.Name, result.Deployed.Hex()))
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
			deployment.Entry.Type == types.LibraryDeployment {
			// For libraries, we just check if the address exists (they might have empty status)
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
		Namespace:      "default", // Libraries always use default namespace
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
