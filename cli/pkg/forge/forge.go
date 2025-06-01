package forge

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Executor handles Forge command execution with enhanced output
type Forge struct {
	projectRoot string
}

// NewExecutor creates a new Forge executor
func NewForge(projectRoot string) *Forge {
	return &Forge{
		projectRoot: projectRoot,
	}
}

// Build runs forge build with proper output handling
func (f *Forge) Build() error {
	cmd := exec.Command("forge", "build")
	cmd.Dir = f.projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Only print error details if build actually failed
		return fmt.Errorf("forge build failed: %w\nOutput: %s", err, string(output))
	}

	// Don't print anything on success - let the caller handle UI
	return nil
}

// CheckForgeInstallation verifies that Forge is installed and accessible
func (f *Forge) CheckInstallation() error {
	cmd := exec.Command("forge", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("forge not found. Please install Foundry: https://getfoundry.sh")
	}
	return nil
}

func (f *Forge) RunScript(scriptPath string, flags []string, envVars map[string]string) (string, error) {
	return f.RunScriptWithArgs(scriptPath, flags, envVars, nil)
}

// RunScriptWithArgs runs a forge script with optional function arguments
func (f *Forge) RunScriptWithArgs(scriptPath string, flags []string, envVars map[string]string, functionArgs []string) (string, error) {
	args := []string{"script", scriptPath}
	
	// Add function arguments BEFORE other flags when using --sig
	// This is important for forge's argument parsing
	if len(functionArgs) > 0 {
		args = append(args, functionArgs...)
	}
	
	args = append(args, flags...)
	
	// Debug: print the full command
	if envVars != nil {
		if _, debug := envVars["DEBUG"]; debug || os.Getenv("TREB_DEBUG") != "" {
			fmt.Printf("Running forge command: forge %s\n", strings.Join(args, " "))
		}
	} else if os.Getenv("TREB_DEBUG") != "" {
		fmt.Printf("Running forge command: forge %s\n", strings.Join(args, " "))
	}
	
	cmd := exec.Command("forge", args...)
	cmd.Dir = f.projectRoot
	cmd.Env = os.Environ()
	for key, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), parseForgeError(err, string(output))
	}
	return string(output), nil
}

// parseForgeError extracts meaningful error messages from forge output
func parseForgeError(err error, output string) error {
	// Check for common error patterns
	if strings.Contains(output, "DeploymentAlreadyExists") {
		re := regexp.MustCompile(`Contract already deployed at: (0x[a-fA-F0-9]{40})`)
		if match := re.FindStringSubmatch(output); len(match) > 1 {
			return fmt.Errorf("contract already deployed at %s", match[1])
		}
		return fmt.Errorf("contract already deployed")
	}

	if strings.Contains(output, "insufficient funds") {
		return fmt.Errorf("insufficient funds for deployment")
	}

	if strings.Contains(output, "nonce too low") {
		return fmt.Errorf("nonce too low - transaction may have already been sent")
	}

	if strings.Contains(output, "replacement transaction underpriced") {
		return fmt.Errorf("replacement transaction underpriced - increase gas price")
	}

	// Check for CreateX collision error
	if strings.Contains(output, "CreateCollision") || strings.Contains(output, "create collision") {
		return fmt.Errorf("contract already exists at this address (CreateX collision).\nThis was most likely deployed but not found in the current deployments.json - make sure you have the latest deployments.json.\nAlternatively, use a different label with --label flag")
	}

	// Extract revert reason
	revertRegex := regexp.MustCompile(`reverted with reason string '([^']+)'`)
	if match := revertRegex.FindStringSubmatch(output); len(match) > 1 {
		return fmt.Errorf("transaction reverted: %s", match[1])
	}

	// Extract any explicit error message
	errorRegex := regexp.MustCompile(`Error:\s+(.+)`)
	if match := errorRegex.FindStringSubmatch(output); len(match) > 1 {
		return fmt.Errorf("%s", match[1])
	}

	// If no specific error found, return the original error with partial output
	if len(output) > 500 {
		return fmt.Errorf("%v\nOutput (last 500 chars): ...%s", err, output[len(output)-500:])
	}
	return fmt.Errorf("%v\nOutput: %s", err, output)
}
