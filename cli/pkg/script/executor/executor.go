package executor

import (
	"fmt"
	"strconv"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/senders"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
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
	Script         *types.ContractInfo
	Network        string
	Namespace      string            // Namespace to use (sets NAMESPACE env var)
	EnvVars        map[string]string // Additional environment variables
	DryRun         bool
	Debug          bool
	DebugJSON      bool
	AdditionalArgs []string
}

// Run executes a Foundry script and returns the raw forge result
func (e *Executor) Run(opts RunOptions) (*forge.ScriptResult, error) {
	senderConfigs, err := e.getSenderConfigs(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender configs: %w", err)
	}

	env, err := e.buildEnvironment(opts, senderConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to build environment: %w", err)
	}

	// Query deployed libraries from registry
	libraries, err := e.getDeployedLibraries(opts.Namespace)
	if err != nil {
		// Log warning but continue - libraries are optional
		fmt.Printf("Warning: Failed to load deployed libraries: %v\n", err)
		libraries = []string{}
	}

	// Convert to forge options
	forgeOpts := forge.ScriptOptions{
		Script:          opts.Script,
		Network:         opts.Network,
		RpcUrl:          e.network.RpcUrl,
		Profile:         opts.Namespace,
		DryRun:          opts.DryRun,
		Broadcast:       !opts.DryRun,
		Debug:           opts.Debug,
		JSON:            !opts.Debug || opts.DebugJSON,
		AdditionalArgs:  opts.AdditionalArgs,
		EnvVars:         env,
		UseLedger:       senders.RequiresLedgerFlag(senderConfigs),
		UseTrezor:       senders.RequiresTrezorFlag(senderConfigs),
		DerivationPaths: senders.GetDerivationPaths(senderConfigs),
		Libraries:       libraries,
	}

	// Execute the script
	return e.forge.Run(forgeOpts)
}

func (e *Executor) getSenderConfigs(opts RunOptions) (*config.SenderConfigs, error) {
	trebConfig, err := config.LoadTrebConfig(e.projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load treb config: %w", err)
	}

	profileTrebConfig, err := trebConfig.GetProfileTrebConfig(opts.Namespace)
	if err != nil {
		return nil, fmt.Errorf("profile not found: %s", opts.Namespace)
	}

	allSenderConfigs, err := config.BuildSenderConfigs(profileTrebConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build sender configs: %w", err)
	}

	// Check if script has sender dependencies
	var senderConfigs *config.SenderConfigs
	if opts.Script.Artifact != nil {
		senderDeps, err := senders.ParseSenderDependencies(opts.Script.Artifact)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sender dependencies: %w", err)
		}

		if len(senderDeps) > 0 {
			// Filter senders based on dependencies
			senderConfigs, err = senders.FilterSenderConfigs(allSenderConfigs, senderDeps, profileTrebConfig.Senders)
			if err != nil {
				return nil, fmt.Errorf("failed to filter sender configs: %w", err)
			}
		} else {
			// No dependencies, use all senders
			senderConfigs = allSenderConfigs
		}
	} else {
		// No artifact, use all senders
		senderConfigs = allSenderConfigs
	}

	return senderConfigs, nil
}

// buildEnvironment builds environment variables for the script
func (e *Executor) buildEnvironment(opts RunOptions, senderConfigs *config.SenderConfigs) (map[string]string, error) {
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
	profileTrebConfig, err := trebConfig.GetProfileTrebConfig(opts.Namespace)
	if err != nil {
		return nil, fmt.Errorf("profile not found: %s", opts.Namespace)
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
	env["FOUNDRY_PROFILE"] = opts.Namespace
	env["DRYRUN"] = strconv.FormatBool(opts.DryRun)

	// Add library deployer if configured
	if profileTrebConfig.LibraryDeployer != "" {
		env["TREB_LIB_DEPLOYER"] = profileTrebConfig.LibraryDeployer
	}

	return env, nil
}

// ExecuteRaw provides direct access to forge script execution
// This is useful for commands that don't need event parsing
func (e *Executor) ExecuteRaw(script *types.ContractInfo, functionSig string, args []string, dryRun bool) (*forge.ScriptResult, error) {
	opts := forge.ScriptOptions{
		Script:       script,
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

// getDeployedLibraries queries the registry for deployed libraries in the given network/namespace
func (e *Executor) getDeployedLibraries(namespace string) ([]string, error) {
	manager, err := registry.NewManager(e.projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry manager: %w", err)
	}

	// Get all deployments for the namespace
	deployments, err := manager.GetDeploymentsByNamespace(namespace, e.network.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to load deployments: %w", err)
	}

	var libraries []string

	// Filter for library deployments
	for _, deployment := range deployments {
		// Check if this is a library deployment
		if deployment.Type == types.LibraryDeployment {
			// Format: "file:lib:address"
			// Extract the artifact path to get the file:lib part
			if deployment.Artifact.Path != "" {
				libRef := fmt.Sprintf("%s:%s:%s", deployment.Artifact.Path, deployment.ContractName, deployment.Address)
				libraries = append(libraries, libRef)
			}
		}
	}

	return libraries, nil
}
