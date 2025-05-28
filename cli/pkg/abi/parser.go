package abi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ABIInput represents a constructor input parameter
type ABIInput struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	InternalType string `json:"internalType"`
}

// ABIConstructor represents the constructor function in an ABI
type ABIConstructor struct {
	Type   string     `json:"type"`
	Inputs []ABIInput `json:"inputs"`
}

// Method represents a function in the ABI
type Method struct {
	Name   string     `json:"name"`
	Type   string     `json:"type"`
	Inputs []ABIInput `json:"inputs"`
}

// ContractABI represents the parsed ABI of a contract
type ContractABI struct {
	Constructor *ABIConstructor
	HasConstructor bool
	Methods []Method
}

// Parser handles ABI parsing from Foundry artifacts
type Parser struct {
	projectRoot string
}

// NewParser creates a new ABI parser
func NewParser(projectRoot string) *Parser {
	return &Parser{
		projectRoot: projectRoot,
	}
}

// ParseContractABI parses the ABI from a contract's artifact file
func (p *Parser) ParseContractABI(contractName string) (*ContractABI, error) {
	artifactPath := filepath.Join(p.projectRoot, "out", fmt.Sprintf("%s.sol", contractName), fmt.Sprintf("%s.json", contractName))
	
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact file %s: %w", artifactPath, err)
	}
	
	var artifact struct {
		ABI []json.RawMessage `json:"abi"`
	}
	
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse artifact JSON: %w", err)
	}
	
	result := &ContractABI{
		HasConstructor: false,
		Methods: []Method{},
	}
	
	// Look for constructor and methods in ABI
	for _, rawABI := range artifact.ABI {
		var abiEntry struct {
			Type   string     `json:"type"`
			Name   string     `json:"name"`
			Inputs []ABIInput `json:"inputs"`
		}
		
		if err := json.Unmarshal(rawABI, &abiEntry); err != nil {
			continue
		}
		
		switch abiEntry.Type {
		case "constructor":
			result.HasConstructor = true
			result.Constructor = &ABIConstructor{
				Type:   abiEntry.Type,
				Inputs: abiEntry.Inputs,
			}
		case "function":
			result.Methods = append(result.Methods, Method{
				Name:   abiEntry.Name,
				Type:   abiEntry.Type,
				Inputs: abiEntry.Inputs,
			})
		}
	}
	
	return result, nil
}

// GenerateConstructorArgs generates Solidity constructor argument code
func (p *Parser) GenerateConstructorArgs(abi *ContractABI) (string, string) {
	if !abi.HasConstructor || len(abi.Constructor.Inputs) == 0 {
		return "", "return \"\";"
	}
	
	var variables []string
	var args []string
	
	for i, input := range abi.Constructor.Inputs {
		varName := input.Name
		if varName == "" {
			varName = fmt.Sprintf("arg%d", i)
		}
		
		// Generate variable declaration with default value
		defaultValue := p.getDefaultValue(input.Type)
		typeDecl := p.getTypeDeclaration(input.Type)
		variables = append(variables, fmt.Sprintf("        %s %s = %s;", typeDecl, varName, defaultValue))
		args = append(args, varName)
	}
	
	variableDecl := strings.Join(variables, "\n")
	encodeCall := fmt.Sprintf("return abi.encode(%s);", strings.Join(args, ", "))
	
	return variableDecl, encodeCall
}

// getTypeDeclaration returns the proper type declaration with memory/storage location
func (p *Parser) getTypeDeclaration(solidityType string) string {
	switch {
	case solidityType == "string":
		return "string memory"
	case solidityType == "bytes":
		return "bytes memory"
	case strings.HasPrefix(solidityType, "bytes") && !strings.Contains(solidityType, "["):
		// Fixed-size bytes (bytes1, bytes32, etc.) don't need memory location
		return solidityType
	case strings.HasSuffix(solidityType, "[]"):
		// Array types need memory location
		return solidityType + " memory"
	default:
		// Value types (uint, int, bool, address) and structs
		return solidityType
	}
}

// getDefaultValue returns a default value for a Solidity type
func (p *Parser) getDefaultValue(solidityType string) string {
	switch {
	case strings.HasPrefix(solidityType, "uint"):
		return "0"
	case strings.HasPrefix(solidityType, "int"):
		return "0"
	case solidityType == "bool":
		return "false"
	case solidityType == "address":
		return "address(0)"
	case solidityType == "string":
		return "\"\""
	case solidityType == "bytes":
		return "\"\""
	case strings.HasPrefix(solidityType, "bytes"):
		return "\"\""
	case strings.HasSuffix(solidityType, "[]"):
		// Array type
		baseType := strings.TrimSuffix(solidityType, "[]")
		return fmt.Sprintf("new %s[](0)", baseType)
	default:
		// For complex types, use zero value
		if strings.Contains(solidityType, "struct") {
			return "/* TODO: Initialize struct */"
		}
		return "/* TODO: Set default value */"
	}
}

// FindInitializeMethod finds an initialize method in the ABI
func (p *Parser) FindInitializeMethod(abi *ContractABI) *Method {
	// Look for common initialize method names
	initializeNames := []string{"initialize", "init", "__init", "initializer"}
	
	for _, method := range abi.Methods {
		for _, initName := range initializeNames {
			if strings.EqualFold(method.Name, initName) {
				return &method
			}
		}
	}
	
	return nil
}

// GenerateInitializerArgs generates Solidity initializer argument code for proxy deployment
func (p *Parser) GenerateInitializerArgs(method *Method) (string, string) {
	if method == nil || len(method.Inputs) == 0 {
		return "", "return \"\";"
	}
	
	var variables []string
	var args []string
	
	for i, input := range method.Inputs {
		varName := input.Name
		if varName == "" {
			varName = fmt.Sprintf("arg%d", i)
		}
		
		// Generate variable declaration with default value
		defaultValue := p.getDefaultValue(input.Type)
		typeDecl := p.getTypeDeclaration(input.Type)
		variables = append(variables, fmt.Sprintf("        %s %s = %s;", typeDecl, varName, defaultValue))
		args = append(args, varName)
	}
	
	variableDecl := strings.Join(variables, "\n")
	
	// For initializer, we need to encode with selector
	selectorCall := fmt.Sprintf("bytes4 selector = bytes4(keccak256(\"%s(%s)\"));", 
		method.Name, 
		p.getMethodSignature(method.Inputs))
	
	encodeCall := fmt.Sprintf("return abi.encodeWithSelector(selector, %s);", strings.Join(args, ", "))
	
	return variableDecl + "\n        \n        " + selectorCall, encodeCall
}

// getMethodSignature generates the method signature string for function selector
func (p *Parser) getMethodSignature(inputs []ABIInput) string {
	var types []string
	for _, input := range inputs {
		types = append(types, p.normalizeType(input.Type))
	}
	return strings.Join(types, ",")
}

// normalizeType normalizes a Solidity type for signature generation
func (p *Parser) normalizeType(solidityType string) string {
	// Remove memory/storage/calldata keywords
	normalized := solidityType
	normalized = strings.ReplaceAll(normalized, " memory", "")
	normalized = strings.ReplaceAll(normalized, " storage", "")
	normalized = strings.ReplaceAll(normalized, " calldata", "")
	
	// Handle fixed-size arrays
	if strings.Contains(normalized, "[") && strings.Contains(normalized, "]") {
		// Keep array notation as-is
		return normalized
	}
	
	return normalized
}