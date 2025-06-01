package script

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
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

// ParameterParser handles parsing of script parameters from natspec
type ParameterParser struct{}

// NewParameterParser creates a new parameter parser
func NewParameterParser() *ParameterParser {
	return &ParameterParser{}
}

// ParseFromArtifact extracts parameters from a contract artifact's natspec
func (p *ParameterParser) ParseFromArtifact(artifact *contracts.Artifact) ([]Parameter, error) {
	// Extract devdoc from metadata
	var devdoc struct {
		Methods map[string]map[string]interface{} `json:"methods"`
	}

	if err := json.Unmarshal(artifact.Metadata.Output.DevDoc, &devdoc); err != nil {
		return nil, fmt.Errorf("failed to parse devdoc: %w", err)
	}

	// Look for run() method
	runMethod, exists := devdoc.Methods["run()"]
	if !exists {
		return nil, nil // No run() method found
	}

	// Look for custom:env tag
	customEnv, exists := runMethod["custom:env"]
	if !exists {
		return nil, nil // No custom:env tag found
	}

	envStr, ok := customEnv.(string)
	if !ok {
		return nil, fmt.Errorf("custom:env is not a string")
	}

	return p.parseCustomEnvString(envStr)
}

// parseCustomEnvString parses the combined custom:env string into parameters
func (p *ParameterParser) parseCustomEnvString(envStr string) ([]Parameter, error) {
	// The compiler combines all entries into a single line
	// Format: {type} name description{type2} name2 description2...
	// We need to split this back into individual entries

	var params []Parameter

	// Regex to match parameter entries
	// Matches: {type} or {type:optional} followed by name and description
	re := regexp.MustCompile(`\{([^}]+)\}\s+(\w+)\s+([^{]+)`)
	matches := re.FindAllStringSubmatch(envStr, -1)

	for _, match := range matches {
		if len(match) != 4 {
			continue
		}

		typeStr := match[1]
		name := match[2]
		description := strings.TrimSpace(match[3])

		// Check if optional
		optional := false
		if strings.HasSuffix(typeStr, ":optional") {
			optional = true
			typeStr = strings.TrimSuffix(typeStr, ":optional")
		}

		// Validate type
		paramType := ParameterType(typeStr)
		if !p.isValidType(paramType) {
			return nil, fmt.Errorf("invalid parameter type: %s", typeStr)
		}

		params = append(params, Parameter{
			Name:        name,
			Type:        paramType,
			Description: description,
			Optional:    optional,
		})
	}

	return params, nil
}

// isValidType checks if a parameter type is valid
func (p *ParameterParser) isValidType(t ParameterType) bool {
	switch t {
	case TypeString, TypeAddress, TypeUint256, TypeInt256, TypeBytes32, TypeBytes,
		TypeSender, TypeDeployment, TypeArtifact:
		return true
	default:
		return false
	}
}

// ValidateValue validates a value against a parameter type
func (p *ParameterParser) ValidateValue(param Parameter, value string) error {
	if value == "" && !param.Optional {
		return fmt.Errorf("parameter %s is required", param.Name)
	}

	if value == "" && param.Optional {
		return nil // Empty optional values are ok
	}

	switch param.Type {
	case TypeString:
		// Any string is valid
		return nil

	case TypeAddress:
		// Basic address validation (should be 0x followed by 40 hex chars)
		if !regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`).MatchString(value) {
			return fmt.Errorf("invalid address format: %s", value)
		}
		return nil

	case TypeUint256, TypeInt256:
		// For now, accept any numeric string or hex string
		// The Solidity contract will do the actual parsing
		if !regexp.MustCompile(`^-?\d+$|^0x[0-9a-fA-F]+$`).MatchString(value) {
			return fmt.Errorf("invalid numeric format: %s", value)
		}
		return nil

	case TypeBytes32:
		// Should be 0x followed by 64 hex chars
		if !regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`).MatchString(value) {
			return fmt.Errorf("invalid bytes32 format: %s", value)
		}
		return nil

	case TypeBytes:
		// Should be 0x followed by even number of hex chars
		if !regexp.MustCompile(`^0x[0-9a-fA-F]*$`).MatchString(value) || len(value)%2 != 0 {
			return fmt.Errorf("invalid bytes format: %s", value)
		}
		return nil

	case TypeSender, TypeDeployment, TypeArtifact:
		// These will be validated by their respective resolvers
		return nil

	default:
		return fmt.Errorf("unknown parameter type: %s", param.Type)
	}
}
