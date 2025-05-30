package script

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
)

// Executor handles running Foundry scripts and parsing their output
type Executor struct {
	projectPath string
	dryRun      bool
	debug       bool
}

// NewExecutor creates a new script executor
func NewExecutor(projectPath string) *Executor {
	return &Executor{
		projectPath: projectPath,
	}
}

// RunOptions contains options for running a script
type RunOptions struct {
	ScriptPath     string
	Network        string
	Profile        string
	DryRun         bool
	Debug          bool
	DebugJSON      bool
	AdditionalArgs []string
}

// RunResult contains the result of running a script
type RunResult struct {
	RawOutput      []byte
	ParsedEvents   []DeploymentEvent  // Legacy: only deployment events
	AllEvents      []ParsedEvent       // New: all event types
	BroadcastPath  string
	Success        bool
}

// Run executes a Foundry script and parses the output
func (e *Executor) Run(opts RunOptions) (*RunResult, error) {
	// Build environment variables
	env, err := e.buildEnvironment(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build environment: %w", err)
	}

	// Build forge script command
	args := e.buildForgeArgs(opts)
	
	// Print command in debug mode
	if opts.Debug {
		fmt.Printf("Running command: forge %s\n", strings.Join(args, " "))
	}

	// Execute the script
	cmd := exec.Command("forge", args...)
	cmd.Dir = e.projectPath
	cmd.Env = append(os.Environ(), env...)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if opts.Debug && !opts.DebugJSON {
		// Also print to console in debug mode (but not in JSON mode)
		cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
		cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)
	}

	err = cmd.Run()
	if err != nil {
		return &RunResult{
			RawOutput: append(stdout.Bytes(), stderr.Bytes()...),
			Success:   false,
		}, fmt.Errorf("forge script failed: %w\nstderr: %s", err, stderr.String())
	}

	// Parse the output
	result := &RunResult{
		RawOutput: stdout.Bytes(),
		Success:   true,
	}

	// Handle output based on debug mode
	if opts.Debug && !opts.DebugJSON {
		// Plain debug mode - don't save to file or attempt parsing
		// Output already printed to console via MultiWriter
		return result, nil
	}

	// If debug JSON mode, save raw output to file and print it
	if opts.DebugJSON {
		debugPath := filepath.Join(e.projectPath, "debug-output.json")
		if err := os.WriteFile(debugPath, result.RawOutput, 0644); err != nil {
			fmt.Printf("Warning: failed to write debug output: %v\n", err)
		} else {
			fmt.Printf("Debug output written to: %s\n", debugPath)
		}
		
		fmt.Printf("\n=== Raw JSON Output ===\n")
		fmt.Print(string(result.RawOutput))
		fmt.Printf("\n=== End Raw JSON Output ===\n")
	}

	// Parse events from the output (only in normal mode or debug-json mode)
	events, allEvents, err := e.parseOutput(result.RawOutput)
	if err != nil {
		if opts.DebugJSON {
			fmt.Printf("Warning: failed to parse events: %v\n", err)
		}
	} else {
		result.ParsedEvents = events
		result.AllEvents = allEvents
	}

	// Find broadcast file if any
	if !opts.DryRun {
		broadcastPath := e.findBroadcastFile(opts.ScriptPath, opts.Network)
		if broadcastPath != "" {
			result.BroadcastPath = broadcastPath
		}
	}

	return result, nil
}

// buildEnvironment builds the environment variables for the script
func (e *Executor) buildEnvironment(opts RunOptions) ([]string, error) {
	env := []string{}

	// Load treb config to get senders
	trebConfig, err := config.LoadTrebConfig(e.projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load treb config: %w", err)
	}

	// Get profile treb configuration
	profileTrebConfig, err := trebConfig.GetProfileTrebConfig(opts.Profile)
	if err != nil {
		return nil, fmt.Errorf("profile not found: %s", opts.Profile)
	}

	// Build sender configs
	senderConfigs, err := BuildSenderConfigs(profileTrebConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build sender configs: %w", err)
	}

	// Encode sender configs
	encodedConfigs, err := EncodeSenderConfigs(senderConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to encode sender configs: %w", err)
	}

	// Add SENDER_CONFIGS
	env = append(env, fmt.Sprintf("SENDER_CONFIGS=%s", encodedConfigs))
	
	// Debug: print sender configs
	if opts.Debug {
		fmt.Printf("SENDER_CONFIGS: %s\n", encodedConfigs)
		// Print sender names
		var senderNames []string
		for _, config := range senderConfigs.Configs {
			senderNames = append(senderNames, config.Name)
		}
		fmt.Printf("Configured senders: %v\n", senderNames)
	}

	// Get network RPC URL from foundry.toml
	foundryConfig, err := config.LoadFoundryConfig(e.projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load foundry config: %w", err)
	}

	// Check if network exists in rpc_endpoints
	rpcURL, ok := foundryConfig.RpcEndpoints[opts.Network]
	if !ok {
		// Fallback to environment variable
		rpcURL = os.Getenv("RPC_URL")
		if rpcURL == "" {
			// Try to get from a standard env var based on network
			networkEnvVar := fmt.Sprintf("%s_RPC_URL", strings.ToUpper(strings.Replace(opts.Network, "-", "_", -1)))
			rpcURL = os.Getenv(networkEnvVar)
			if rpcURL == "" {
				return nil, fmt.Errorf("RPC URL not found for network: %s (not in foundry.toml rpc_endpoints or environment)", opts.Network)
			}
		}
	}
	
	// Add NETWORK env var (expected by Dispatcher for creating forks)
	env = append(env, fmt.Sprintf("NETWORK=%s", rpcURL))

	// Add NAMESPACE (default for now)
	namespace := "default"
	if ns := os.Getenv("DEPLOYMENT_NAMESPACE"); ns != "" {
		namespace = ns
	}
	env = append(env, fmt.Sprintf("NAMESPACE=%s", namespace))

	// Add DRYRUN flag
	if opts.DryRun {
		env = append(env, "DRYRUN=true")
	}

	return env, nil
}

// buildForgeArgs builds the forge script command arguments
func (e *Executor) buildForgeArgs(opts RunOptions) []string {
	args := []string{"script"}

	// Add script path
	args = append(args, opts.ScriptPath)

	// Get network RPC URL from foundry.toml
	foundryConfig, err := config.LoadFoundryConfig(e.projectPath)
	if err == nil {
		// Check if network exists in rpc_endpoints
		if rpcURL, ok := foundryConfig.RpcEndpoints[opts.Network]; ok {
			args = append(args, "--rpc-url", rpcURL)
		} else {
			// Fallback to environment variable
			rpcURL := os.Getenv("RPC_URL")
			if rpcURL == "" {
				// Try to get from a standard env var based on network
				networkEnvVar := fmt.Sprintf("%s_RPC_URL", strings.ToUpper(strings.Replace(opts.Network, "-", "_", -1)))
				rpcURL = os.Getenv(networkEnvVar)
			}
			if rpcURL != "" {
				args = append(args, "--rpc-url", rpcURL)
			}
		}
	}

	// Add broadcast flag if not dry run
	if !opts.DryRun {
		args = append(args, "--broadcast")
	}

	// Add JSON output flag only when NOT in plain debug mode
	if !opts.Debug || opts.DebugJSON {
		// Add JSON flag for normal mode or debug-json mode
		args = append(args, "--json")
	}
	// Plain debug mode (--debug only) will not add --json flag

	// Add verbosity for better error messages
	args = append(args, "-vvvv")

	// Add any additional arguments
	args = append(args, opts.AdditionalArgs...)

	return args
}

// parseOutput parses the forge script output to extract events
func (e *Executor) parseOutput(output []byte) ([]DeploymentEvent, []ParsedEvent, error) {
	// Parse the forge output
	forgeOutput, err := ParseForgeOutput(output)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse forge output: %w", err)
	}

	// Extract all events from the parsed output
	allEvents, err := ParseAllEvents(forgeOutput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse all events: %w", err)
	}

	// Extract deployment events for legacy compatibility
	deploymentEvents, err := ParseEvents(forgeOutput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse deployment events: %w", err)
	}

	return deploymentEvents, allEvents, nil
}

// findBroadcastFile finds the broadcast file for the executed script
func (e *Executor) findBroadcastFile(scriptPath, network string) string {
	// Broadcast files are typically in broadcast/<ScriptName>/<chainId>/run-latest.json
	// TODO: Get chain ID from network mapping
	// For now, we'll just return empty
	return ""
}