package executor

import (
	"fmt"
	"strconv"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
)

// Executor handles script execution using the forge package
type Executor struct {
	projectPath string
	network     *network.NetworkInfo
	forge       *forge.Forge
}

// NewExecutor creates a new script executor
func NewExecutor(projectPath string, network *network.NetworkInfo) *Executor {
	return &Executor{
		projectPath: projectPath,
		network:     network,
		forge:       forge.NewForge(projectPath),
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

// Run executes a Foundry script and returns the raw forge result
func (e *Executor) Run(opts RunOptions) (*forge.ScriptResult, error) {
	// Convert to forge options
	forgeOpts := forge.ScriptOptions{
		ScriptPath:     opts.ScriptPath,
		Network:        opts.Network,
		RpcUrl:         e.network.RpcUrl,
		Profile:        opts.Profile,
		DryRun:         opts.DryRun,
		Broadcast:      !opts.DryRun,
		Debug:          opts.Debug,
		JSON:           !opts.Debug || opts.DebugJSON,
		AdditionalArgs: opts.AdditionalArgs,
	}

	// Build environment variables
	env, err := e.buildEnvironment(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build environment: %w", err)
	}
	forgeOpts.EnvVars = env

	// Execute the script
	return e.forge.Run(forgeOpts)
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
	senderConfigs, err := config.BuildSenderConfigs(profileTrebConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build sender configs: %w", err)
	}

	// Encode sender configs
	encodedConfigs, err := config.EncodeSenderConfigs(senderConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to encode sender configs: %w", err)
	}

	// Add core environment variables
	env["SENDER_CONFIGS"] = encodedConfigs
	env["NAMESPACE"] = opts.Namespace
	env["NETWORK"] = e.network.Name
	env["FOUNDRY_PROFILE"] = opts.Profile
	env["DRYRUN"] = strconv.FormatBool(opts.DryRun)

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
