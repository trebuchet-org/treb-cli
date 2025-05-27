package deployment

import (
	"fmt"
	"path/filepath"

	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

type DeploymentParams struct {
	DeploymentType      types.DeploymentType
	ContractQuery       string
	ImplementationQuery string
	TargetQuery         string
	Namespace           string
	Label               string
	NetworkName         string
	Sender              string
	Predict             bool
	Debug               bool
}

// DeploymentContext holds all deployment configuration
type DeploymentContext struct {
	// Config
	Params      DeploymentParams
	projectRoot string
	// Services
	generator       *contracts.Generator
	registryManager *registry.Manager
	forge           *forge.Forge
	// Deployment
	ScriptPath           string
	envVars              map[string]string
	contractInfo         *contracts.ContractInfo
	implementationInfo   *contracts.ContractInfo
	networkInfo          *network.NetworkInfo
	targetDeploymentFQID string
	resolvedLibraries    []LibraryInfo
}

// NewDeploymentContext creates a new deployment context with explicit registry manager
func NewDeploymentContext(projectRoot string, params *DeploymentParams, registryManager *registry.Manager) *DeploymentContext {
	return &DeploymentContext{
		Params:          *params,
		projectRoot:     projectRoot,
		envVars:         make(map[string]string),
		registryManager: registryManager,
		generator:       contracts.NewGenerator(projectRoot),
		forge:           forge.NewForge(projectRoot),
	}
}

// NewContext creates a new deployment context
func NewContext(params DeploymentParams) (*DeploymentContext, error) {
	projectRoot := "."
	
	registryPath := filepath.Join(projectRoot, "deployments.json")
	registryManager, err := registry.NewManager(registryPath)
	if err != nil {
		return nil, err
	}
	
	ctx := NewDeploymentContext(projectRoot, &params, registryManager)

	networkResolver := network.NewResolver(projectRoot)
	networkInfo, err := networkResolver.ResolveNetwork(ctx.Params.NetworkName)
	if err != nil {
		return nil, err
	}
	ctx.networkInfo = networkInfo

	forgeExecutor := forge.NewForge(".")
	ctx.forge = forgeExecutor

	return ctx, nil
}

// GetIdentifier returns the deployment identifier based on type
func (ctx *DeploymentContext) GetShortID() string {
	var identifier string
	switch ctx.Params.DeploymentType {
	case types.ProxyDeployment:
		// For proxies, use the implementation name + "Proxy"
		if ctx.implementationInfo != nil {
			identifier = ctx.implementationInfo.Name + "Proxy"
		} else {
			// Fallback to contract name if implementation info not available
			identifier = ctx.contractInfo.Name
		}
	default:
		identifier = ctx.contractInfo.Name
	}

	if ctx.Params.Label != "" {
		identifier += ":" + ctx.Params.Label
	}

	return identifier
}

// GetFullIdentifier returns the full deployment identifier including environment and label
func (ctx *DeploymentContext) GetFQID() string {
	// Always use contractInfo.Path as it represents the actual contract being deployed
	// For proxies, this is the proxy contract path, not the implementation
	return fmt.Sprintf("%d/%s/%s:%s", ctx.networkInfo.ChainID(), ctx.Params.Namespace, ctx.contractInfo.Path, ctx.GetShortID())
}
