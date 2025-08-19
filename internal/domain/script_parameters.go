package domain

// ParameterType represents the type of a script parameter
type ParameterType string

const (
	ParamTypeString     ParameterType = "string"
	ParamTypeAddress    ParameterType = "address"
	ParamTypeUint256    ParameterType = "uint256"
	ParamTypeInt256     ParameterType = "int256"
	ParamTypeBytes32    ParameterType = "bytes32"
	ParamTypeBytes      ParameterType = "bytes"
	ParamTypeBool       ParameterType = "bool"
	ParamTypeSender     ParameterType = "sender"
	ParamTypeDeployment ParameterType = "deployment"
	ParamTypeArtifact   ParameterType = "artifact"
)

// ScriptParameter represents a parameter expected by a script
type ScriptParameter struct {
	Name        string
	Type        ParameterType
	Description string
	Optional    bool
}

// ScriptParameterValue represents a resolved parameter value
type ScriptParameterValue struct {
	Name  string
	Type  ParameterType
	Value string
}
