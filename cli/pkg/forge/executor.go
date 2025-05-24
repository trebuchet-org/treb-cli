package forge

import (
	"fmt"
	"os/exec"
	"strings"
)

// Executor handles Forge command execution with enhanced output
type Executor struct {
	projectRoot string
}

// NewExecutor creates a new Forge executor
func NewExecutor(projectRoot string) *Executor {
	return &Executor{
		projectRoot: projectRoot,
	}
}

// Build runs forge build with proper output handling
func (e *Executor) Build() error {
	cmd := exec.Command("forge", "build")
	cmd.Dir = e.projectRoot
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Only print error details if build actually failed
		return fmt.Errorf("forge build failed: %w\nOutput: %s", err, string(output))
	}
	
	// Don't print anything on success - let the caller handle UI
	return nil
}

// RunScript runs a forge script with enhanced output and error handling
func (e *Executor) RunScript(scriptName, networkName string, broadcast bool) error {
	args := []string{
		"script",
		fmt.Sprintf("script/%s.s.sol", scriptName),
		"--rpc-url", networkName,
		"-vvvv", // High verbosity for better error messages
	}
	
	if broadcast {
		args = append(args, "--broadcast")
	}
	
	// Don't print execution details - let caller handle UI
	
	cmd := exec.Command("forge", args...)
	cmd.Dir = e.projectRoot
	
	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	
	if err != nil {
		fmt.Printf("❌ Script execution failed:\n")
		fmt.Printf("Command: forge %s\n", strings.Join(args, " "))
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Full output:\n%s\n", outputStr)
		return fmt.Errorf("forge script failed: %w", err)
	}
	
	// Parse successful output for key information
	e.parseScriptOutput(outputStr)
	
	return nil
}

// parseScriptOutput extracts key information from forge script output
func (e *Executor) parseScriptOutput(output string) {
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Look for transaction hash
		if strings.Contains(line, "Transaction hash:") {
			fmt.Printf("🔍 %s\n", line)
		}
		
		// Look for contract address
		if strings.Contains(line, "Contract Address:") {
			fmt.Printf("📍 %s\n", line)
		}
		
		// Look for gas used
		if strings.Contains(line, "Gas used:") {
			fmt.Printf("⛽ %s\n", line)
		}
		
		// Look for block number
		if strings.Contains(line, "Block:") {
			fmt.Printf("📊 %s\n", line)
		}
	}
}

// CheckForgeInstallation verifies that Forge is installed and accessible
func (e *Executor) CheckForgeInstallation() error {
	cmd := exec.Command("forge", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("forge not found. Please install Foundry: https://getfoundry.sh")
	}
	return nil
}