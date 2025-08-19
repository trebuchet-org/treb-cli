package environment

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// BuilderAdapter adapts environment building for script execution
type BuilderAdapter struct {
	projectPath string
}

// NewBuilderAdapter creates a new environment builder adapter
func NewBuilderAdapter(projectPath string) *BuilderAdapter {
	return &BuilderAdapter{
		projectPath: projectPath,
	}
}

// BuildEnvironment builds the complete environment for script execution
func (b *BuilderAdapter) BuildEnvironment(
	ctx context.Context,
	params usecase.BuildEnvironmentParams,
) (map[string]string, error) {
	env := make(map[string]string)

	// Add all resolved parameters
	for k, v := range params.Parameters {
		env[k] = v
	}

	// Encode sender configs
	encodedConfigs, err := b.encodeSenderConfigs(params.TrebConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to encode sender configs: %w", err)
	}

	// Add core environment variables (matching v1 executor)
	env["SENDER_CONFIGS"] = encodedConfigs
	env["NAMESPACE"] = params.Namespace
	env["NETWORK"] = params.Network
	// Use namespace for profile, default to "default" if empty
	profile := params.Namespace
	if profile == "" {
		profile = "default"
	}
	env["FOUNDRY_PROFILE"] = profile
	env["DRYRUN"] = strconv.FormatBool(params.DryRun)

	// Add library deployer if configured
	if params.TrebConfig != nil && params.TrebConfig.LibraryDeployer != "" {
		env["TREB_LIB_DEPLOYER"] = params.TrebConfig.LibraryDeployer
	}

	// Add deployed libraries if any
	if len(params.DeployedLibraries) > 0 {
		env["DEPLOYED_LIBRARIES"] = b.encodeLibraries(params.DeployedLibraries)
	}

	return env, nil
}

// encodeSenderConfigs encodes sender configs to the format expected by Foundry scripts
func (b *BuilderAdapter) encodeSenderConfigs(trebConfig *domain.TrebConfig) (string, error) {
	// Use the proper ABI encoder
	return ABIEncodeSenderConfigs(trebConfig)
}

// encodeLibraries encodes library references for the environment
func (b *BuilderAdapter) encodeLibraries(libraries []usecase.LibraryReference) string {
	// Format: "file:lib:address" separated by spaces
	// This matches what the forge executor expects
	var encoded []string
	for _, lib := range libraries {
		ref := fmt.Sprintf("%s:%s:%s", lib.Path, lib.Name, lib.Address)
		encoded = append(encoded, ref)
	}
	return strings.Join(encoded, " ")
}

