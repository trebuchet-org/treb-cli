package render

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// AnvilRenderer renders anvil operation results
type AnvilRenderer struct{}

// NewAnvilRenderer creates a new anvil renderer
func NewAnvilRenderer() *AnvilRenderer {
	return &AnvilRenderer{}
}

// Render renders the anvil operation result
func (r *AnvilRenderer) Render(result *usecase.ManageAnvilResult) error {
	switch result.Operation {
	case "start":
		return r.renderStart(result)
	case "stop":
		return r.renderStop(result)
	case "restart":
		return r.renderRestart(result)
	case "status":
		return r.renderStatus(result)
	default:
		return fmt.Errorf("unknown operation: %s", result.Operation)
	}
}

// renderStart renders the start operation result
func (r *AnvilRenderer) renderStart(result *usecase.ManageAnvilResult) error {
	if result.Success {
		color.New(color.FgGreen).Printf("✅ %s\n", result.Message)
		color.New(color.FgYellow).Printf("📋 Logs: %s\n", result.Status.LogFile)
		color.New(color.FgBlue).Printf("🌐 RPC URL: %s\n", result.Status.RPCURL)
		
		if result.Status.CreateXDeployed {
			color.New(color.FgGreen).Printf("✅ CreateX factory deployed at %s\n", result.Status.CreateXAddress)
		} else {
			color.New(color.FgRed).Printf("⚠️  Warning: Failed to deploy CreateX\n")
			color.New(color.FgYellow).Println("Deployments may fail without CreateX factory")
		}
	}
	return nil
}

// renderStop renders the stop operation result
func (r *AnvilRenderer) renderStop(result *usecase.ManageAnvilResult) error {
	if result.Success {
		color.New(color.FgGreen).Printf("✅ %s\n", result.Message)
	}
	return nil
}

// renderRestart renders the restart operation result
func (r *AnvilRenderer) renderRestart(result *usecase.ManageAnvilResult) error {
	return r.renderStart(result)
}

// renderStatus renders the status operation result
func (r *AnvilRenderer) renderStatus(result *usecase.ManageAnvilResult) error {
	color.New(color.FgCyan, color.Bold).Printf("📊 Anvil Status ('%s'):\n", result.Instance.Name)
	
	if result.Status.Running {
		color.New(color.FgGreen).Printf("Status: 🟢 Running (PID %d)\n", result.Status.PID)
		color.New(color.FgBlue).Printf("RPC URL: %s\n", result.Status.RPCURL)
		color.New(color.FgYellow).Printf("Log file: %s\n", result.Status.LogFile)
		
		if result.Status.RPCHealthy {
			color.New(color.FgGreen).Println("RPC Health: ✅ Responding")
		} else {
			color.New(color.FgRed).Println("RPC Health: ❌ Not responding")
		}
		
		if result.Status.CreateXDeployed {
			color.New(color.FgGreen).Printf("CreateX Status: ✅ Deployed at %s\n", result.Status.CreateXAddress)
		} else {
			color.New(color.FgRed).Println("CreateX Status: ❌ Not deployed")
		}
	} else {
		color.New(color.FgRed).Println("Status: 🔴 Not running")
		color.New(color.FgHiBlack).Printf("PID file: %s\n", result.Instance.PidFile)
		color.New(color.FgHiBlack).Printf("Log file: %s\n", result.Instance.LogFile)
	}
	
	return nil
}

// RenderLogsHeader renders the header for logs streaming
func (r *AnvilRenderer) RenderLogsHeader(result *usecase.ManageAnvilResult) error {
	color.New(color.FgCyan, color.Bold).Printf("📋 Showing anvil '%s' logs (Ctrl+C to exit):\n", result.Instance.Name)
	color.New(color.FgHiBlack).Printf("Log file: %s\n\n", result.Status.LogFile)
	return nil
}