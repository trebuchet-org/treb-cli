package domain

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
	Type            ScriptType
	ContractName    string
	ArtifactPath    string
	Strategy        ScriptDeploymentStrategy
	ProxyInfo       *ScriptProxyInfo // nil for non-proxy deployments
	ConstructorInfo *ConstructorInfo // nil if no constructor
	ScriptPath      string
}

// ScriptProxyInfo contains proxy-specific deployment information for script generation
type ScriptProxyInfo struct {
	ProxyName       string
	ProxyPath       string
	ProxyArtifact   string
	InitializerInfo *InitializerInfo
}

// ConstructorInfo contains constructor parameter information
type ConstructorInfo struct {
	HasConstructor bool
	Parameters     []Parameter
}

// InitializerInfo contains initializer method information
type InitializerInfo struct {
	MethodName string
	Parameters []Parameter
}

// Parameter represents a function parameter
type Parameter struct {
	Name         string
	Type         string
	InternalType string
}

// ContractABI represents the parsed ABI of a contract
type ContractABI struct {
	Name           string
	HasConstructor bool
	Constructor    *Constructor
	Methods        []Method
}

// Constructor represents the constructor in an ABI
type Constructor struct {
	Inputs []Parameter
}

// Method represents a function in the ABI
type Method struct {
	Name   string
	Inputs []Parameter
}