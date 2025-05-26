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

// Type represents the type of deployment
type Type string

const (
	TypeSingleton Type = "singleton"
	TypeProxy     Type = "proxy"
	TypeLibrary   Type = "library"
)

type DeploymentParams struct {
	DeploymentType      Type
	ContractQuery       string
	ImplementationQuery string
	Env                 string
	Label               string
	NetworkName         string
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
	ScriptPath   string
	envVars      map[string]string
	contractInfo *contracts.ContractInfo
	networkInfo  *network.NetworkInfo
	// Result
	Deployment *types.DeploymentResult
}

// NewContext creates a new deployment context
func NewContext(params DeploymentParams) (*DeploymentContext, error) {
	ctx := &DeploymentContext{
		Params:      params,
		envVars:     make(map[string]string),
		projectRoot: ".",
	}

	registryPath := filepath.Join(".", "deployments.json")
	registryManager, err := registry.NewManager(registryPath)
	if err != nil {
		return nil, err
	}
	ctx.registryManager = registryManager

	generator := contracts.NewGenerator(".")
	ctx.generator = generator

	networkResolver := network.NewResolver(".")
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
	case TypeProxy:
		identifier = ctx.contractInfo.Name + "Proxy"
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
	return fmt.Sprintf("%d/%s/%s:%s", ctx.networkInfo.ChainID(), ctx.Params.Env, ctx.contractInfo.Path, ctx.GetShortID())
}
