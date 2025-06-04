package script

import (
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/treb"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
)

// Executor is a refactored version using the forge package for script execution
type Executor struct {
	projectPath string
	network     *network.NetworkInfo
	forge       *forge.Forge
	parser      *forge.EventParser
}

// NewExecutor creates a new script executor
func NewExecutor(projectPath string, network *network.NetworkInfo) *Executor {
	return &Executor{
		projectPath: projectPath,
		network:     network,
		forge:       forge.NewForge(projectPath),
		parser:      forge.NewEventParser(),
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
	ParsedEvents  []*treb.TrebContractDeployed // Contract deployment events using generated types
	AllEvents     []interface{}                // New: all event types (using generated types)
	Logs          []string                     // Console.log output from the script
	BroadcastPath string
	Success       bool
}

// Run executes a Foundry script and returns structured results
func (e *Executor) Run(opts RunOptions) (*RunResult, error) {
	// Convert to forge options
	forgeOpts := forge.ScriptOptions{
		ScriptPath:     opts.ScriptPath,
		Network:        opts.Network,
		RpcUrl:         e.network.RpcUrl,
		Profile:        opts.Profile,
		DryRun:         opts.DryRun,
		Broadcast:      !opts.DryRun,
		Debug:          opts.Debug,
		JSON:           true, // Always use JSON for structured parsing
		AdditionalArgs: opts.AdditionalArgs,
	}

	// Build environment variables
	env, err := e.buildEnvironment(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build environment: %w", err)
	}
	forgeOpts.EnvVars = env

	// Execute the script
	scriptResult, err := e.forge.Run(forgeOpts)
	if err != nil {
		return nil, err
	}

	// Build result
	result := &RunResult{
		RawOutput: scriptResult.RawOutput,
		Success:   scriptResult.Success,
	}

	// Handle errors
	if scriptResult.Error != nil {
		return result, scriptResult.Error
	}

	// Parse events if we have output
	if scriptResult.ParsedOutput != nil && scriptResult.ParsedOutput.ScriptOutput != nil {
		// Parse all events
		allEvents, err := e.parser.ParseEvents(scriptResult.ParsedOutput.ScriptOutput)
		if err != nil {
			// Don't fail on parse errors, just log them
			if opts.Debug || opts.DebugJSON {
				fmt.Printf("Warning: failed to parse events: %v\n", err)
			}
		}

		result.AllEvents = allEvents
		result.ParsedEvents = forge.ExtractDeploymentEvents(allEvents)
		result.Logs = scriptResult.ParsedOutput.ConsoleLogs
	}

	// Set broadcast path
	result.BroadcastPath = scriptResult.BroadcastPath

	return result, nil
}

// buildEnvironment builds environment variables for the script
func (e *Executor) buildEnvironment(opts RunOptions) (map[string]string, error) {
	env := make(map[string]string)

	// Copy additional env vars
	for k, v := range opts.EnvVars {
		env[k] = v
	}

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

	// Add core environment variables
	env["SENDER_CONFIGS"] = encodedConfigs
	env["NAMESPACE"] = opts.Namespace
	env["NETWORK"] = e.network.Name
	env["FOUNDRY_PROFILE"] = opts.Profile

	// Add library deployer if configured
	if profileTrebConfig.LibraryDeployer != "" {
		env["TREB_LIB_DEPLOYER"] = profileTrebConfig.LibraryDeployer
	}

	return env, nil
}

// ExecuteRaw provides direct access to forge script execution
// This is useful for commands that don't need event parsing
func (e *Executor) ExecuteRaw(scriptPath string, functionSig string, args []string, dryRun bool) (*forge.ScriptResult, error) {
	opts := forge.ScriptOptions{
		ScriptPath:   scriptPath,
		FunctionName: functionSig,
		FunctionArgs: args,
		Network:      e.network.Name,
		RpcUrl:       e.network.RpcUrl,
		DryRun:       dryRun,
		Broadcast:    !dryRun,
		JSON:         true,
	}

	return e.forge.Run(opts)
}

// ParseEventsFromOutput parses events from raw forge output
// This is useful when you have output from another source
func (e *Executor) ParseEventsFromOutput(output []byte) ([]interface{}, []*treb.TrebContractDeployed, error) {
	// Parse the output
	parsed, err := e.forge.ParseOutput(output)
	if err != nil {
		return nil, nil, err
	}

	if parsed.ScriptOutput == nil {
		return nil, nil, fmt.Errorf("no script output found")
	}

	// Parse events
	allEvents, err := e.parser.ParseEvents(parsed.ScriptOutput)
	if err != nil {
		return nil, nil, err
	}

	// Extract deployment events
	deploymentEvents := forge.ExtractDeploymentEvents(allEvents)

	return allEvents, deploymentEvents, nil
}
