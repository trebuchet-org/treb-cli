package deployment

import (
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// Type represents the type of deployment
type Type string

const (
	TypeSingleton Type = "singleton"
	TypeProxy     Type = "proxy" 
	TypeLibrary   Type = "library"
)

// Context holds all deployment configuration
type Context struct {
	Type                Type
	ContractName        string
	ProxyName           string
	ImplementationName  string
	ImplementationLabel string
	Env                 string
	Label               string
	Predict             bool
	Debug               bool
	Verify              bool
	NetworkName         string
	NetworkInfo         *network.NetworkInfo
	EnvVars             map[string]string
	ScriptPath          string
	Deployment          *types.DeploymentResult
}

// NewContext creates a new deployment context
func NewContext(deployType Type) *Context {
	return &Context{
		Type:    deployType,
		EnvVars: make(map[string]string),
	}
}

// GetIdentifier returns the deployment identifier based on type
func (ctx *Context) GetIdentifier() string {
	switch ctx.Type {
	case TypeProxy:
		return ctx.ProxyName
	case TypeLibrary:
		return ctx.ContractName
	default:
		return ctx.ContractName
	}
}

// GetFullIdentifier returns the full deployment identifier including environment and label
func (ctx *Context) GetFullIdentifier() string {
	identifier := ctx.GetIdentifier()
	
	if ctx.Type != TypeLibrary && ctx.Env != "" {
		identifier = ctx.Env + "/" + identifier
	}
	
	if ctx.Label != "" {
		identifier += ":" + ctx.Label
	}
	
	return identifier
}