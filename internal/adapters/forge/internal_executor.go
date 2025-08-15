package forge

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// InternalForgeExecutor handles Forge command execution without pkg dependencies
type InternalForgeExecutor struct {
	projectRoot string
}

// NewInternalForgeExecutor creates a new internal forge executor
func NewInternalForgeExecutor(projectRoot string) *InternalForgeExecutor {
	return &InternalForgeExecutor{
		projectRoot: projectRoot,
	}
}

// Build runs forge build with proper output handling
func (f *InternalForgeExecutor) Build() error {
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

// CheckInstallation verifies that Forge is installed and accessible
func (f *InternalForgeExecutor) CheckInstallation() error {
	cmd := exec.Command("forge", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("forge not found. Please install Foundry: https://getfoundry.sh")
	}
	return nil
}

// RunScript runs a forge script
func (f *InternalForgeExecutor) RunScript(scriptPath string, flags []string, envVars map[string]string) (string, error) {
	return f.RunScriptWithArgs(scriptPath, flags, envVars, nil)
}

// RunScriptWithArgs runs a forge script with optional function arguments
func (f *InternalForgeExecutor) RunScriptWithArgs(scriptPath string, flags []string, envVars map[string]string, functionArgs []string) (string, error) {
	args := []string{"script", scriptPath, "--ffi"}

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
		return string(output), parseInternalForgeError(err, string(output))
	}
	return string(output), nil
}

// parseInternalForgeError extracts meaningful error messages from forge output
func parseInternalForgeError(err error, output string) error {
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

	if strings.Contains(output, "execution reverted") {
		// Extract revert reason if available
		re := regexp.MustCompile(`Error: ([^\n]+)`)
		if match := re.FindStringSubmatch(output); len(match) > 1 {
			return fmt.Errorf("execution reverted: %s", match[1])
		}
		return fmt.Errorf("execution reverted")
	}

	if strings.Contains(output, "Transaction dropped from the mempool") {
		return fmt.Errorf("transaction dropped from mempool (possible nonce issue)")
	}

	// Return the original error with output
	return fmt.Errorf("%w\nOutput: %s", err, output)
}