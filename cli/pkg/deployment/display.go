package deployment

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// PrintSummary prints the deployment summary header
func (d *DeploymentContext) PrintSummary() {
	// Clear any previous output
	fmt.Println()

	// Determine action
	action := "Deploying"
	if d.Params.Predict {
		action = "Predicting address for"
	}

	// Build identifier
	identifier := d.GetShortID()

	// Print header
	color.New(color.FgCyan, color.Bold).Printf("%s ", action)
	color.New(color.FgWhite, color.Bold).Printf("%s ", identifier)
	if d.networkInfo != nil {
		color.New(color.FgCyan).Printf("to ")
		color.New(color.FgMagenta, color.Bold).Printf("%s\n\n", d.networkInfo.Name)
	} else {
		fmt.Println()
	}
}

func (d *DeploymentContext) printExistingDeployment(deployment *types.DeploymentEntry) {
	fmt.Println()
	color.New(color.FgYellow, color.Bold).Printf("🔄 Deployment Already Exists\n")
	color.New(color.FgWhite, color.Bold).Printf("Address:      ")
	color.New(color.FgGreen, color.Bold).Printf("%s\n", deployment.Address.Hex())
	color.New(color.FgWhite, color.Bold).Printf("Network:      ")
	color.New(color.FgMagenta).Printf("%s\n", d.networkInfo.Name)
	color.New(color.FgWhite, color.Bold).Printf("Transaction:  ")
	color.New(color.FgMagenta).Printf("%s\n", deployment.Deployment.TxHash.Hex())
	fmt.Println()
}

// ShowSuccess shows deployment success details
func (d *DeploymentContext) ShowSuccess(result *ParsedDeploymentResult) {
	fmt.Println()
	color.New(color.FgGreen, color.Bold).Printf("🚀 Deployment Successful!\n")
	fmt.Println()

	// Contract info
	d.printContractInfo(result)

	// Network info
	d.printNetworkInfo(result)

	// Transaction info (if available)
	if result.TxHash != (common.Hash{}) {
		d.printTransactionInfo(result)
	}

	// Safe info (if pending)
	if result.SafeTxHash != ([32]byte{}) {
		d.printSafeInfo(result)
	}

	d.printVerificationInfo(result)

	fmt.Println()
}

// ShowPrediction shows predicted deployment address
func (d *DeploymentContext) ShowPrediction(result *ParsedDeploymentResult) {
	fmt.Println()
	color.New(color.FgGreen, color.Bold).Printf("🎯 Predicted Deployment Address\n")
	fmt.Println()

	// Contract info
	switch d.Params.DeploymentType {
	case types.SingletonDeployment:
		color.New(color.FgWhite, color.Bold).Printf("Contract:     ")
		fmt.Printf("%s", d.GetShortID())
	case types.ProxyDeployment:
		color.New(color.FgWhite, color.Bold).Printf("Proxy:        ")
		fmt.Printf("%s", d.GetShortID())
		if d.contractInfo != nil {
			fmt.Printf(" (%s)", d.contractInfo.Name)
		}
	case types.LibraryDeployment:
		color.New(color.FgWhite, color.Bold).Printf("Library:      ")
		fmt.Printf("%s", d.contractInfo.Name)
	}
	fmt.Println()

	// Network info
	color.New(color.FgWhite, color.Bold).Printf("Network:      ")
	color.New(color.FgMagenta).Printf("%s\n", d.networkInfo.Name)

	// Address
	color.New(color.FgWhite, color.Bold).Printf("Address:      ")
	color.New(color.FgGreen, color.Bold).Printf("%s\n", result.Predicted.Hex())

	// Salt (if available)
	if result.Salt != (common.Hash{}) {
		color.New(color.FgWhite, color.Bold).Printf("Salt:         ")
		fmt.Printf("%x\n", result.Salt)
	}

	fmt.Println()
}

// CreateSpinner creates a new spinner with the given message
func (d *DeploymentContext) CreateSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	s.Color("cyan", "bold")
	s.Start()
	return s
}

// PrintStep prints a step completion message
func (d *DeploymentContext) PrintStep(message string) {
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("%s\n", message)
}

// PrintError prints an error message
func (d *DeploymentContext) PrintError(err error) {
	color.New(color.FgRed).Printf("✗ ")
	fmt.Printf("Error: %v\n", err)
}

// Helper methods

func (d *DeploymentContext) printContractInfo(result *ParsedDeploymentResult) {
	switch d.Params.DeploymentType {
	case types.SingletonDeployment:
		color.New(color.FgWhite, color.Bold).Printf("Contract:     ")
		fmt.Printf("%s", d.GetShortID())
		fmt.Println()

		color.New(color.FgWhite, color.Bold).Printf("Address:      ")
		color.New(color.FgGreen, color.Bold).Printf("%s\n", result.Deployed.Hex())

	case types.ProxyDeployment:
		color.New(color.FgWhite, color.Bold).Printf("Proxy:        ")
		fmt.Printf("%s", d.GetShortID())
		fmt.Println()

		color.New(color.FgWhite, color.Bold).Printf("Address:      ")
		color.New(color.FgGreen, color.Bold).Printf("%s\n", result.Deployed.Hex())

		// Show proxy contract details
		if d.contractInfo != nil {
			color.New(color.FgWhite, color.Bold).Printf("Type:         ")
			fmt.Printf("%s\n", d.contractInfo.Name)
		}

		// Show implementation details if available
		if d.targetDeploymentFQID != "" {
			color.New(color.FgWhite, color.Bold).Printf("Implementation: ")
			fmt.Printf("%s\n", d.targetDeploymentFQID)
			if implAddress := d.envVars["IMPLEMENTATION_ADDRESS"]; implAddress != "" {
				color.New(color.FgWhite, color.Bold).Printf("Impl Address: ")
				color.New(color.FgBlue).Printf("%s\n", implAddress)
			}
		}

	case types.LibraryDeployment:
		color.New(color.FgWhite, color.Bold).Printf("Library:      ")
		fmt.Printf("%s\n", d.contractInfo.Name)

		color.New(color.FgWhite, color.Bold).Printf("Address:      ")
		color.New(color.FgGreen, color.Bold).Printf("%s\n", result.Deployed.Hex())
	}
}

func (d *DeploymentContext) printNetworkInfo(result *ParsedDeploymentResult) {
	color.New(color.FgWhite, color.Bold).Printf("Network:      ")
	color.New(color.FgMagenta).Printf("%s", d.networkInfo.Name)
	if d.networkInfo.ChainID() > 0 {
		fmt.Printf(" (chain ID: %d)", d.networkInfo.ChainID())
	}
	fmt.Println()
}

func (d *DeploymentContext) printTransactionInfo(result *ParsedDeploymentResult) {
	color.New(color.FgWhite, color.Bold).Printf("Transaction:  ")
	fmt.Printf("%s\n", result.TxHash.Hex())

	if result.BlockNumber > 0 {
		color.New(color.FgWhite, color.Bold).Printf("Block:        ")
		fmt.Printf("%d\n", result.BlockNumber)
	}

	// TODO: Add deployer address when available in result
}

func (d *DeploymentContext) printSafeInfo(result *ParsedDeploymentResult) {
	fmt.Println()
	color.New(color.FgYellow, color.Bold).Printf("⏳ Pending Safe Execution\n")

	color.New(color.FgWhite, color.Bold).Printf("Safe Tx Hash: ")
	fmt.Printf("%s\n", common.Hash(result.SafeTxHash).Hex())

	fmt.Println()
	color.New(color.FgYellow).Printf("Next steps:\n")
	fmt.Printf("1. Collect signatures from Safe owners\n")
	fmt.Printf("2. Execute the transaction through the Safe interface\n")
	fmt.Printf("3. Run 'treb sync' to update the deployment status\n")
}

func (d *DeploymentContext) printVerificationInfo(result *ParsedDeploymentResult) {
	// TODO: Add verification status and explorer URL when available
	fmt.Println()
	color.New(color.FgWhite).Printf("Contract deployed. Run 'treb verify' to verify on block explorer.\n")
}
