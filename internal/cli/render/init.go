package render

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// InitRenderer renders init command results
type InitRenderer struct{}

// NewInitRenderer creates a new init renderer
func NewInitRenderer() *InitRenderer {
	return &InitRenderer{}
}

// Render renders the init project result
func (r *InitRenderer) Render(result *usecase.InitProjectResult) error {
	// Show steps
	for _, step := range result.Steps {
		if step.Success {
			if step.Message != "" {
				color.New(color.FgGreen).Printf("✅ %s\n", step.Message)
			} else {
				color.New(color.FgGreen).Printf("✅ %s\n", step.Name)
			}
		} else {
			color.New(color.FgRed).Printf("❌ %s\n", step.Name)
			if step.Message != "" {
				fmt.Printf("   %s\n", step.Message)
			}
			if step.Error != nil {
				fmt.Printf("   %s\n", step.Error.Error())
			}
		}
	}

	// Show final status
	if result.FoundryProjectValid && result.TrebSolInstalled && (result.RegistryCreated || result.AlreadyInitialized) {
		r.printSuccessMessage(result)
	}

	return nil
}

func (r *InitRenderer) printSuccessMessage(result *usecase.InitProjectResult) {
	fmt.Println("")
	if result.AlreadyInitialized {
		color.New(color.FgYellow).Println("⚠️  treb was already initialized in this project")
	} else {
		color.New(color.FgGreen, color.Bold).Println("🎉 treb initialized successfully!")
	}

	fmt.Println("")
	color.New(color.FgCyan, color.Bold).Println("📋 Next steps:")

	fmt.Println("1. Copy .env.example to .env and configure your deployment keys:")
	fmt.Println("   • Set DEPLOYER_PRIVATE_KEY for your deployment wallet")
	fmt.Println("   • Set RPC URLs for networks you'll deploy to")
	fmt.Println("   • Set API keys for contract verification")
	fmt.Println("")

	fmt.Println("2. Configure deployment environments in treb.toml:")
	fmt.Println("   • Add [ns.<namespace>.senders.<name>] sections for each environment")
	fmt.Println("   • See documentation for Safe multisig and hardware wallet support")
	fmt.Println("")

	fmt.Println("3. Generate your first deployment script:")
	color.New(color.FgHiBlack).Println("   treb gen deploy Counter")
	fmt.Println("")

	fmt.Println("4. Predict and deploy:")
	color.New(color.FgHiBlack).Println("   treb deploy predict Counter --network sepolia")
	color.New(color.FgHiBlack).Println("   treb deploy Counter --network sepolia")
	fmt.Println("")

	fmt.Println("5. View and manage deployments:")
	color.New(color.FgHiBlack).Println("   treb list")
	color.New(color.FgHiBlack).Println("   treb show Counter")
	color.New(color.FgHiBlack).Println("   treb tag Counter v1.0.0")
}
