package deployment

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// Display handles deployment UI output
type Display struct{}

// NewDisplay creates a new deployment display handler
func NewDisplay() *Display {
	return &Display{}
}

// PrintSummary prints the deployment summary header
func (d *Display) PrintSummary(ctx *Context) {
	// Clear any previous output
	fmt.Println()

	// Determine action
	action := "Deploying"
	if ctx.Predict {
		action = "Predicting address for"
	}

	// Build identifier
	identifier := ctx.GetFullIdentifier()

	// Print header
	color.New(color.FgCyan, color.Bold).Printf("%s ", action)
	color.New(color.FgWhite, color.Bold).Printf("%s ", identifier)
	if ctx.NetworkInfo != nil {
		color.New(color.FgCyan).Printf("to ")
		color.New(color.FgMagenta, color.Bold).Printf("%s\n\n", ctx.NetworkInfo.Name)
	} else {
		fmt.Println()
	}
}

// ShowSuccess shows deployment success details
func (d *Display) ShowSuccess(ctx *Context, result *types.DeploymentResult) {
	fmt.Println()
	color.New(color.FgGreen, color.Bold).Printf("🚀 Deployment Successful!\n")
	fmt.Println()

	// Contract info
	d.printContractInfo(ctx, result)

	// Network info
	d.printNetworkInfo(ctx, result)

	// Transaction info (if available)
	if result.TxHash != (common.Hash{}) {
		d.printTransactionInfo(result)
	}

	// Safe info (if pending)
	if result.SafeTxHash != (common.Hash{}) {
		d.printSafeInfo(result)
	}

	d.printVerificationInfo(ctx, result)

	fmt.Println()
}

// ShowPrediction shows predicted deployment address
func (d *Display) ShowPrediction(ctx *Context, predicted *types.PredictResult) {
	fmt.Println()
	color.New(color.FgGreen, color.Bold).Printf("🎯 Predicted Deployment Address\n")
	fmt.Println()

	// Contract info
	switch ctx.Type {
	case TypeSingleton:
		color.New(color.FgWhite, color.Bold).Printf("Contract:     ")
		fmt.Printf("%s/%s", ctx.Env, ctx.ContractInfo.Name)
		if ctx.Label != "" {
			color.New(color.FgCyan).Printf(":%s", ctx.Label)
		}
	case TypeProxy:
		color.New(color.FgWhite, color.Bold).Printf("Proxy:        ")
		fmt.Printf("%s/%s", ctx.Env, ctx.ProxyName)
		if ctx.Label != "" {
			color.New(color.FgCyan).Printf(":%s", ctx.Label)
		}
	case TypeLibrary:
		color.New(color.FgWhite, color.Bold).Printf("Library:      ")
		fmt.Printf("%s", ctx.ContractInfo.Name)
	}
	fmt.Println()

	// Network info
	color.New(color.FgWhite, color.Bold).Printf("Network:      ")
	color.New(color.FgMagenta).Printf("%s\n", ctx.NetworkInfo.Name)

	// Address
	color.New(color.FgWhite, color.Bold).Printf("Address:      ")
	color.New(color.FgGreen, color.Bold).Printf("%s\n", predicted.Address.Hex())

	// Salt (if available)
	if predicted.Salt != [32]byte{} {
		color.New(color.FgWhite, color.Bold).Printf("Salt:         ")
		fmt.Printf("%x\n", predicted.Salt)
	}

	fmt.Println()
}

// CreateSpinner creates a new spinner with the given message
func (d *Display) CreateSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	s.Color("cyan", "bold")
	s.Start()
	return s
}

// PrintStep prints a step completion message
func (d *Display) PrintStep(message string) {
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("%s\n", message)
}

// PrintError prints an error message
func (d *Display) PrintError(err error) {
	color.New(color.FgRed).Printf("✗ ")
	fmt.Printf("Error: %v\n", err)
}

// Helper methods

func (d *Display) printContractInfo(ctx *Context, result *types.DeploymentResult) {
	switch ctx.Type {
	case TypeSingleton:
		color.New(color.FgWhite, color.Bold).Printf("Contract:     ")
		fmt.Printf("%s", ctx.ContractInfo.Name)
		if ctx.Env != "" {
			color.New(color.FgCyan).Printf(" (%s", ctx.Env)
			if ctx.Label != "" {
				fmt.Printf(":%s", ctx.Label)
			}
			fmt.Printf(")")
		}
		fmt.Println()

		color.New(color.FgWhite, color.Bold).Printf("Address:      ")
		color.New(color.FgGreen, color.Bold).Printf("%s\n", result.Address.Hex())

	case TypeProxy:
		color.New(color.FgWhite, color.Bold).Printf("Proxy:        ")
		fmt.Printf("%s", ctx.ProxyName)
		if ctx.Env != "" {
			color.New(color.FgCyan).Printf(" (%s", ctx.Env)
			if ctx.Label != "" {
				fmt.Printf(":%s", ctx.Label)
			}
			fmt.Printf(")")
		}
		fmt.Println()

		color.New(color.FgWhite, color.Bold).Printf("Address:      ")
		color.New(color.FgGreen, color.Bold).Printf("%s\n", result.Address.Hex())

		// TODO: Add implementation address when available in result

	case TypeLibrary:
		color.New(color.FgWhite, color.Bold).Printf("Library:      ")
		fmt.Printf("%s\n", ctx.ContractInfo.Name)

		color.New(color.FgWhite, color.Bold).Printf("Address:      ")
		color.New(color.FgGreen, color.Bold).Printf("%s\n", result.Address.Hex())
	}
}

func (d *Display) printNetworkInfo(ctx *Context, result *types.DeploymentResult) {
	color.New(color.FgWhite, color.Bold).Printf("Network:      ")
	color.New(color.FgMagenta).Printf("%s", ctx.NetworkInfo.Name)
	if ctx.NetworkInfo.ChainID > 0 {
		fmt.Printf(" (chain ID: %d)", ctx.NetworkInfo.ChainID)
	}
	fmt.Println()
}

func (d *Display) printTransactionInfo(result *types.DeploymentResult) {
	color.New(color.FgWhite, color.Bold).Printf("Transaction:  ")
	fmt.Printf("%s\n", result.TxHash.Hex())

	if result.BlockNumber > 0 {
		color.New(color.FgWhite, color.Bold).Printf("Block:        ")
		fmt.Printf("%d\n", result.BlockNumber)
	}

	// TODO: Add deployer address when available in result
}

func (d *Display) printSafeInfo(result *types.DeploymentResult) {
	fmt.Println()
	color.New(color.FgYellow, color.Bold).Printf("⏳ Pending Safe Execution\n")

	color.New(color.FgWhite, color.Bold).Printf("Safe Tx Hash: ")
	fmt.Printf("%s\n", result.SafeTxHash.Hex())

	fmt.Println()
	color.New(color.FgYellow).Printf("Next steps:\n")
	fmt.Printf("1. Collect signatures from Safe owners\n")
	fmt.Printf("2. Execute the transaction through the Safe interface\n")
	fmt.Printf("3. Run 'treb sync' to update the deployment status\n")
}

func (d *Display) printVerificationInfo(ctx *Context, result *types.DeploymentResult) {
	// TODO: Add verification status and explorer URL when available
	fmt.Println()
	color.New(color.FgWhite).Printf("Contract deployed. Run 'treb verify' to verify on block explorer.\n")
}
