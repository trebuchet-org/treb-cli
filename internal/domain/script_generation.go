package domain

import "github.com/ethereum/go-ethereum/accounts/abi"

// ScriptType represents the type of deployment script
type ScriptType string

const (
	ScriptTypeContract ScriptType = "contract"
	ScriptTypeLibrary  ScriptType = "library"
	ScriptTypeProxy    ScriptType = "proxy"
)

// ScriptDeploymentStrategy represents the CREATE opcode strategy for script generation
type ScriptDeploymentStrategy string

const (
	StrategyCreate2 ScriptDeploymentStrategy = "CREATE2"
	StrategyCreate3 ScriptDeploymentStrategy = "CREATE3"
)

// ScriptTemplate contains all information needed to generate a deployment script
type ScriptTemplate struct {
	Type         ScriptType
	ContractName string
	ArtifactPath string
	ScriptPath   string
	Strategy     ScriptDeploymentStrategy
	ABI          *abi.ABI
	ProxyInfo    *ScriptProxyInfo // nil for non-proxy deployments
}

// ScriptProxyInfo contains proxy-specific deployment information for script generation
type ScriptProxyInfo struct {
	ProxyName     string
	ProxyPath     string
	ProxyArtifact string
}
