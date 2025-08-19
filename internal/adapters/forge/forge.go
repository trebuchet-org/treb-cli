package forge

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Forge handles Forge command execution with enhanced output
type Forge struct {
	projectRoot string
}

// NewForge creates a new Forge executor
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

// CheckInstallation verifies that Forge is installed and accessible
func (f *Forge) CheckInstallation() error {
	cmd := exec.Command("forge", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("forge not found. Please install Foundry: https://getfoundry.sh")
	}
	return nil
}

// RunScript runs a forge script
func (f *Forge) RunScript(scriptPath string, flags []string, envVars map[string]string) (string, error) {
	return f.RunScriptWithArgs(scriptPath, flags, envVars, nil)
}

// RunScriptWithArgs runs a forge script with optional function arguments
func (f *Forge) RunScriptWithArgs(scriptPath string, flags []string, envVars map[string]string, functionArgs []string) (string, error) {
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
	fmt.Println("Here?")
	if _, debug := envVars["DEBUG"]; debug || os.Getenv("TREB_DEBUG") != "" {
		fmt.Printf("Forge output: %s\n", string(output))
	} else if os.Getenv("TREB_DEBUG") != "" {
		fmt.Printf("Forge output: %s\n", string(output))
	}

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

	if strings.Contains(output, "reverted") {
		// Try to extract revert reason
		re := regexp.MustCompile(`revert: (.+)`)
		if match := re.FindStringSubmatch(output); len(match) > 1 {
			return fmt.Errorf("transaction reverted: %s", match[1])
		}
		return fmt.Errorf("transaction reverted")
	}

	// Return the original error with output context
	return fmt.Errorf("%w\nforge output: %s", err, output)
}

// InstallDependency installs a Foundry dependency
func (f *Forge) InstallDependency(dependency string) error {
	cmd := exec.Command("forge", "install", dependency, "--no-commit")
	cmd.Dir = f.projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install %s: %w\nOutput: %s", dependency, err, string(output))
	}

	return nil
}

// GetInstalledDependencies returns a list of installed dependencies
func (f *Forge) GetInstalledDependencies() ([]string, error) {
	libPath := f.projectRoot + "/lib"
	entries, err := os.ReadDir(libPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var deps []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			deps = append(deps, entry.Name())
		}
	}

	return deps, nil
}

// Format runs forge fmt on the project
func (f *Forge) Format() error {
	cmd := exec.Command("forge", "fmt")
	cmd.Dir = f.projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("forge fmt failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// Test runs forge test
func (f *Forge) Test(testPattern string) error {
	args := []string{"test"}
	if testPattern != "" {
		args = append(args, "--match-test", testPattern)
	}

	cmd := exec.Command("forge", args...)
	cmd.Dir = f.projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("forge test failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Print(string(output))
	return nil
}
