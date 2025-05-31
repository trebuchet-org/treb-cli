package script

import (
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/treb"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
	ColorBold   = "\033[1m"
)

// TransactionInfo groups related transaction events using generated types
type TransactionInfo struct {
	TransactionID     string
	Simulated         *treb.TrebTransactionSimulated
	BroadcastStarted  *treb.TrebBroadcastStarted     // Marks the start of broadcast phase
	Broadcast         *treb.TrebTransactionBroadcast
	Deployments       []*treb.TrebContractDeployed
	Failed            *treb.TrebTransactionFailed
	ProxyEvents       []interface{} // Upgraded, AdminChanged, BeaconUpgraded events (still from events package)
	SafeQueued        *treb.TrebSafeTransactionQueued // Track if this is a Safe transaction
}

// GetEventIconForGenerated returns an icon for generated event types
func GetEventIconForGenerated(event interface{}) string {
	switch event.(type) {
	case *treb.TrebDeployingContract:
		return "🔨"
	case *treb.TrebContractDeployed:
		return "✅"
	case *treb.TrebSafeTransactionQueued:
		return "🔐"
	case *treb.TrebBroadcastStarted:
		return "🚀"
	case *treb.TrebTransactionBroadcast:
		return "📤"
	case *treb.TrebTransactionSimulated:
		return "🔍"
	case *treb.TrebTransactionFailed:
		return "❌"
	case *events.UpgradedEvent:
		return "⬆️"
	case *events.AdminChangedEvent:
		return "👤"
	case *events.BeaconUpgradedEvent:
		return "🔆"
	default:
		return "📝"
	}
}

// GetEventIcon returns an icon for the event type (legacy support for events package)
func GetEventIcon(eventType events.EventType) string {
	switch eventType {
	case events.EventTypeUpgraded:
		return "⬆️"
	case events.EventTypeAdminChanged:
		return "👤"
	case events.EventTypeBeaconUpgraded:
		return "🔆"
	case events.EventTypeUnknown:
		return "📝"
	default:
		return "📝"
	}
}

// ColorizeContractName returns a colorized contract name
func ColorizeContractName(name string) string {
	return fmt.Sprintf("%s%s%s", ColorCyan, name, ColorReset)
}

// ColorizeAddress returns a colorized address string
func ColorizeAddress(address string) string {
	return fmt.Sprintf("%s%s%s", ColorGray, address, ColorReset)
}

// ColorizeHash returns a colorized hash string (shortened)
func ColorizeHash(hash []byte) string {
	hashStr := fmt.Sprintf("%x", hash[:4])
	return fmt.Sprintf("%s%s%s", ColorGray, hashStr, ColorReset)
}

// FormatGeneratedDeploymentSummary formats a deployment summary using generated types
func FormatGeneratedDeploymentSummary(deployment *treb.TrebContractDeployed, contractName string) string {
	name := ColorizeContractName(contractName)
	address := fmt.Sprintf("%s%s%s", ColorGreen, deployment.Location.Hex(), ColorReset)
	salt := ColorizeHash(deployment.Deployment.Salt[:])
	
	return fmt.Sprintf("%s at %s (salt: %s)", name, address, salt)
}

// PrintDeploymentBanner prints a colored banner for deployment operations
func PrintDeploymentBanner(title string, network string, profile string) {
	fmt.Printf("\n%s%s%s%s\n", ColorBold, ColorCyan, title, ColorReset)
	fmt.Printf("Network: %s%s%s\n", ColorBlue, network, ColorReset)
	fmt.Printf("Profile: %s%s%s\n", ColorBlue, profile, ColorReset)
	fmt.Println()
}

// PrintSuccessMessage prints a success message with green color
func PrintSuccessMessage(message string) {
	fmt.Printf("%s✅ %s%s\n", ColorGreen, message, ColorReset)
}

// PrintWarningMessage prints a warning message with yellow color
func PrintWarningMessage(message string) {
	fmt.Printf("%s⚠️  %s%s\n", ColorYellow, message, ColorReset)
}

// PrintErrorMessage prints an error message with red color
func PrintErrorMessage(message string) {
	fmt.Printf("%s❌ %s%s\n", ColorRed, message, ColorReset)
}

// PrintInfoMessage prints an info message with blue color
func PrintInfoMessage(message string) {
	fmt.Printf("ℹ️  %s%s%s\n", ColorBlue, message, ColorReset)
}