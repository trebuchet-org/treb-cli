package parameters

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ParameterType represents the type of a script parameter
type ParameterType string

const (
	// Base types
	TypeString  ParameterType = "string"
	TypeAddress ParameterType = "address"
	TypeUint256 ParameterType = "uint256"
	TypeInt256  ParameterType = "int256"
	TypeBytes32 ParameterType = "bytes32"
	TypeBytes   ParameterType = "bytes"
	TypeBool    ParameterType = "bool"

	// Meta types
	TypeSender     ParameterType = "sender"
	TypeDeployment ParameterType = "deployment"
	TypeArtifact   ParameterType = "artifact"
)

// Parameter represents a script parameter parsed from natspec
type Parameter struct {
	Name        string
	Type        ParameterType
	Description string
	Optional    bool
}

// Parser handles parsing of script parameters from natspec
type Parser struct {
	projectRoot string
}

// NewParser creates a new parameter parser
func NewParser(projectRoot string) *Parser {
	return &Parser{
		projectRoot: projectRoot,
	}
}

// ParseFromArtifact extracts parameters from a contract artifact's natspec
func (p *Parser) ParseFromArtifact(artifactPath string) ([]Parameter, error) {
	// Read artifact
	data, err := os.ReadFile(filepath.Join(p.projectRoot, artifactPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact: %w", err)
	}

	// Parse artifact
	var artifact struct {
		Metadata string `json:"metadata"`
	}
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse artifact: %w", err)
	}

	if artifact.Metadata == "" {
		return nil, nil
	}

	// Parse metadata
	var metadata struct {
		Output struct {
			DevDoc struct {
				Params map[string]string `json:"params"`
				Custom map[string]string `json:"custom"`
			} `json:"devdoc"`
		} `json:"output"`
	}
	if err := json.Unmarshal([]byte(artifact.Metadata), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Extract parameters from custom params
	var params []Parameter
	if paramsJSON, ok := metadata.Output.DevDoc.Custom["params"]; ok {
		if err := json.Unmarshal([]byte(paramsJSON), &params); err == nil {
			return params, nil
		}
	}

	// Fall back to parsing from devdoc params
	return p.parseFromDevDoc(metadata.Output.DevDoc.Params), nil
}

// parseFromDevDoc parses parameters from devdoc natspec comments
func (p *Parser) parseFromDevDoc(devdoc map[string]string) []Parameter {
	var params []Parameter
	
	// Regex to parse parameter comments like "@param name Description"
	paramRegex := regexp.MustCompile(`^(\w+)\s+(.+)$`)
	
	for name, desc := range devdoc {
		matches := paramRegex.FindStringSubmatch(desc)
		if len(matches) > 0 {
			param := Parameter{
				Name:        name,
				Description: strings.TrimSpace(desc),
				Type:        p.inferType(name, desc),
				Optional:    strings.Contains(strings.ToLower(desc), "optional"),
			}
			params = append(params, param)
		}
	}
	
	return params
}

// inferType tries to infer parameter type from name and description
func (p *Parser) inferType(name, desc string) ParameterType {
	lowerName := strings.ToLower(name)
	lowerDesc := strings.ToLower(desc)
	
	// Check for explicit types in description
	if strings.Contains(lowerDesc, "address") {
		return TypeAddress
	}
	if strings.Contains(lowerDesc, "uint256") || strings.Contains(lowerDesc, "number") {
		return TypeUint256
	}
	if strings.Contains(lowerDesc, "bool") {
		return TypeBool
	}
	if strings.Contains(lowerDesc, "bytes32") {
		return TypeBytes32
	}
	if strings.Contains(lowerDesc, "bytes") {
		return TypeBytes
	}
	
	// Check common naming patterns
	if strings.HasSuffix(lowerName, "address") || lowerName == "to" || lowerName == "from" {
		return TypeAddress
	}
	if strings.Contains(lowerName, "amount") || strings.Contains(lowerName, "value") {
		return TypeUint256
	}
	if strings.HasPrefix(lowerName, "is") || strings.HasPrefix(lowerName, "has") {
		return TypeBool
	}
	if lowerName == "sender" || lowerName == "deployer" {
		return TypeSender
	}
	if strings.Contains(lowerName, "deployment") {
		return TypeDeployment
	}
	if strings.Contains(lowerName, "artifact") {
		return TypeArtifact
	}
	
	// Default to string
	return TypeString
}

// ExtractFromRunFunction extracts parameters from the run function signature
func (p *Parser) ExtractFromRunFunction(scriptPath string) ([]Parameter, error) {
	// Read script file
	content, err := os.ReadFile(filepath.Join(p.projectRoot, scriptPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read script: %w", err)
	}
	
	// Find run function
	runFuncRegex := regexp.MustCompile(`function\s+run\s*\(([^)]*)\)`)
	matches := runFuncRegex.FindStringSubmatch(string(content))
	if len(matches) < 2 {
		return nil, nil // No parameters
	}
	
	// Parse parameters
	paramStr := matches[1]
	if strings.TrimSpace(paramStr) == "" {
		return nil, nil
	}
	
	var params []Parameter
	paramParts := strings.Split(paramStr, ",")
	
	for _, part := range paramParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		// Parse "type name" format
		tokens := strings.Fields(part)
		if len(tokens) >= 2 {
			paramType := p.solidityTypeToParamType(tokens[0])
			paramName := tokens[1]
			
			params = append(params, Parameter{
				Name: paramName,
				Type: paramType,
			})
		}
	}
	
	return params, nil
}

// solidityTypeToParamType converts Solidity types to parameter types
func (p *Parser) solidityTypeToParamType(solType string) ParameterType {
	switch solType {
	case "address":
		return TypeAddress
	case "uint256", "uint":
		return TypeUint256
	case "int256", "int":
		return TypeInt256
	case "bytes32":
		return TypeBytes32
	case "bytes":
		return TypeBytes
	case "bool":
		return TypeBool
	case "string":
		return TypeString
	default:
		return TypeString
	}
}