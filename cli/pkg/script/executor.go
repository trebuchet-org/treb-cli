package script

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
)

// Executor handles running Foundry scripts and parsing their output
type Executor struct {
	projectPath string
	network     *network.NetworkInfo
}

// NewExecutor creates a new script executor
func NewExecutor(projectPath string, network *network.NetworkInfo) *Executor {
	return &Executor{
		projectPath: projectPath,
		network:     network,
	}
}

// RunOptions contains options for running a script
type RunOptions struct {
	ScriptPath     string
	Network        string
	Profile        string
	Namespace      string            // Namespace to use (sets NAMESPACE env var)
	EnvVars        map[string]string // Additional environment variables
	DryRun         bool
	Debug          bool
	DebugJSON      bool
	AdditionalArgs []string
}

// RunResult contains the result of running a script
type RunResult struct {
	RawOutput     []byte
	ParsedEvents  []DeploymentEvent // Legacy: only deployment events
	AllEvents     []ParsedEvent     // New: all event types
	BroadcastPath string
	Success       bool
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
		// Create a unique run directory
		runUUID := fmt.Sprintf("%d", time.Now().Unix())
		debugDir := filepath.Join(e.projectPath, "out", ".treb-debug", runUUID)
		if err := os.MkdirAll(debugDir, 0755); err != nil {
			fmt.Printf("Warning: failed to create debug directory: %v\n", err)
		} else {
			// Save raw output
			rawPath := filepath.Join(debugDir, "raw-output.json")
			if err := os.WriteFile(rawPath, result.RawOutput, 0644); err != nil {
				fmt.Printf("Warning: failed to write raw output: %v\n", err)
			}

			// Parse and save parsed/ignored output
			parsedPath := filepath.Join(debugDir, "parsed-output.json")
			ignoredPath := filepath.Join(debugDir, "ignored-output.txt")
			e.debugParseOutput(result.RawOutput, parsedPath, ignoredPath)

			fmt.Printf("Debug output written to: %s\n", debugDir)
			fmt.Printf("  - raw-output.json: Complete forge output\n")
			fmt.Printf("  - parsed-output.json: Successfully parsed JSON objects\n")
			fmt.Printf("  - ignored-output.txt: Lines that were not parsed\n")
		}

		// Also save to legacy location for compatibility
		debugPath := filepath.Join(e.projectPath, "debug-output.json")
		if err := os.WriteFile(debugPath, result.RawOutput, 0644); err != nil {
			fmt.Printf("Warning: failed to write legacy debug output: %v\n", err)
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

	// Add NAMESPACE
	namespace := opts.Namespace
	if namespace == "" {
		// Fallback to environment variable
		namespace = os.Getenv("DEPLOYMENT_NAMESPACE")
		if namespace == "" {
			namespace = "default"
		}
	}
	env = append(env, fmt.Sprintf("NAMESPACE=%s", namespace))

	// Add DRYRUN flag
	if opts.DryRun {
		env = append(env, "DRYRUN=true")
	}

	// Add any custom environment variables
	for key, value := range opts.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Debug: print environment info
	if opts.Debug {
		fmt.Printf("NAMESPACE: %s\n", namespace)
		if len(opts.EnvVars) > 0 {
			fmt.Println("Custom environment variables:")
			for k, v := range opts.EnvVars {
				fmt.Printf("  %s=%s\n", k, v)
			}
		}
	}

	return env, nil
}

// buildForgeArgs builds the forge script command arguments
func (e *Executor) buildForgeArgs(opts RunOptions) []string {
	args := []string{"script"}

	// Add script path
	args = append(args, opts.ScriptPath)
	args = append(args, "--rpc-url", e.network.RpcUrl)

	// Add broadcast flag if not dry run and not debug mode
	if !opts.DryRun && !opts.Debug {
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
func (e *Executor) findBroadcastFile(scriptPath, networkName string) string {
	// Get script name from path
	scriptName := filepath.Base(scriptPath)

	// Resolve network to get actual chain ID from RPC
	resolver := network.NewResolver(e.projectPath)
	networkInfo, err := resolver.ResolveNetwork(networkName)
	if err != nil {
		// Can't determine chain ID, so can't find broadcast file
		return ""
	}

	chainIDStr := fmt.Sprintf("%d", networkInfo.ChainID)

	// Look for broadcast file
	broadcastPath := filepath.Join(e.projectPath, "broadcast", scriptName, chainIDStr, "run-latest.json")
	if _, err := os.Stat(broadcastPath); err == nil {
		return broadcastPath
	}

	// Check without extension
	scriptNameNoExt := strings.TrimSuffix(scriptName, filepath.Ext(scriptName))
	broadcastPath = filepath.Join(e.projectPath, "broadcast", scriptNameNoExt, chainIDStr, "run-latest.json")
	if _, err := os.Stat(broadcastPath); err == nil {
		return broadcastPath
	}

	return ""
}

// debugParseOutput parses the forge output and separates parsed vs ignored lines
func (e *Executor) debugParseOutput(output []byte, parsedPath, ignoredPath string) {
	var parsedObjects []json.RawMessage
	var ignoredLines []string

	// Use ParseCompleteForgeOutput to try to parse all JSON objects
	parsedOutput, _ := ParseCompleteForgeOutput(output)

	// Collect successfully parsed objects
	if parsedOutput != nil {
		if parsedOutput.ScriptOutput != nil {
			if data, err := json.Marshal(parsedOutput.ScriptOutput); err == nil {
				parsedObjects = append(parsedObjects, json.RawMessage(data))
			}
		}
		if parsedOutput.GasEstimate != nil {
			if data, err := json.Marshal(parsedOutput.GasEstimate); err == nil {
				parsedObjects = append(parsedObjects, json.RawMessage(data))
			}
		}
		if parsedOutput.StatusOutput != nil {
			if data, err := json.Marshal(parsedOutput.StatusOutput); err == nil {
				parsedObjects = append(parsedObjects, json.RawMessage(data))
			}
		}
	}

	// Now scan through the output line by line to find what was ignored
	scanner := bufio.NewScanner(bytes.NewReader(output))
	const maxTokenSize = 200 * 1024 * 1024 // 200MB
	buf := make([]byte, maxTokenSize)
	scanner.Buffer(buf, maxTokenSize)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this line is a JSON object that we could parse
		if strings.HasPrefix(strings.TrimSpace(line), "{") {
			var testJSON json.RawMessage
			if err := json.Unmarshal([]byte(line), &testJSON); err == nil {
				// It's valid JSON, check if we already parsed it
				wasParsed := false

				// Check against our known parsed types
				var forgeOut ForgeScriptOutput
				var gasEst GasEstimate
				var statusOut StatusOutput

				if err := json.Unmarshal([]byte(line), &forgeOut); err == nil && forgeOut.RawLogs != nil {
					wasParsed = true
				} else if err := json.Unmarshal([]byte(line), &gasEst); err == nil && gasEst.Chain != 0 {
					wasParsed = true
				} else if err := json.Unmarshal([]byte(line), &statusOut); err == nil && statusOut.Status != "" {
					wasParsed = true
				}

				if !wasParsed {
					// This is a JSON object we didn't parse
					parsedObjects = append(parsedObjects, json.RawMessage(line))
					ignoredLines = append(ignoredLines, line)
				}
			} else {
				// Not valid JSON but starts with {
				ignoredLines = append(ignoredLines, line)
			}
		} else {
			// Non-JSON line
			ignoredLines = append(ignoredLines, line)
		}
	}

	// Write parsed objects to file
	if len(parsedObjects) > 0 {
		parsedData, _ := json.MarshalIndent(map[string]interface{}{
			"parsed_objects": parsedObjects,
			"count":          len(parsedObjects),
		}, "", "  ")
		os.WriteFile(parsedPath, parsedData, 0644)
	}

	// Write ignored lines to file
	if len(ignoredLines) > 0 {
		ignoredData := []byte(strings.Join(ignoredLines, "\n"))
		os.WriteFile(ignoredPath, ignoredData, 0644)
	}
}
