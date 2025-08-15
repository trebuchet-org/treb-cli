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
	Constructor    *ABIConstructor
	HasConstructor bool
	Methods        []Method
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

// ParseContractABI parses the ABI for a contract from its artifact
func (p *Parser) ParseContractABI(contractName string) (*ContractABI, error) {
	// Try to find the artifact
	artifactPath := p.findArtifact(contractName)
	if artifactPath == "" {
		return nil, fmt.Errorf("artifact not found for contract %s", contractName)
	}

	// Read the artifact
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact: %w", err)
	}

	// Parse the JSON
	var artifact struct {
		ABI json.RawMessage `json:"abi"`
	}
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse artifact JSON: %w", err)
	}

	// Parse the ABI
	var abiData []json.RawMessage
	if err := json.Unmarshal(artifact.ABI, &abiData); err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	result := &ContractABI{
		Methods: []Method{},
	}

	// Process each ABI entry
	for _, entry := range abiData {
		var base struct {
			Type string `json:"type"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(entry, &base); err != nil {
			continue
		}

		switch base.Type {
		case "constructor":
			var constructor ABIConstructor
			if err := json.Unmarshal(entry, &constructor); err != nil {
				continue
			}
			result.Constructor = &constructor
			result.HasConstructor = true

		case "function":
			var method Method
			if err := json.Unmarshal(entry, &method); err != nil {
				continue
			}
			result.Methods = append(result.Methods, method)
		}
	}

	return result, nil
}

// FindInitializeMethod finds an initializer method in the ABI
func (p *Parser) FindInitializeMethod(abi *ContractABI) *Method {
	if abi == nil {
		return nil
	}

	// Look for common initializer method names
	initNames := []string{"initialize", "init", "initializer"}
	
	for _, method := range abi.Methods {
		for _, initName := range initNames {
			if strings.EqualFold(method.Name, initName) {
				return &method
			}
		}
	}

	return nil
}

// GenerateConstructorArgs generates variable declarations and encoding for constructor arguments
func (p *Parser) GenerateConstructorArgs(abi *ContractABI) (vars string, encode string) {
	if abi == nil || abi.Constructor == nil || len(abi.Constructor.Inputs) == 0 {
		return "", ""
	}

	return p.generateArgs(abi.Constructor.Inputs, "constructor")
}

// GenerateInitializerArgs generates variable declarations and encoding for initializer arguments
func (p *Parser) GenerateInitializerArgs(method *Method) (vars string, encode string) {
	if method == nil || len(method.Inputs) == 0 {
		return "", ""
	}

	return p.generateArgs(method.Inputs, "initializer")
}

// generateArgs generates variable declarations and encoding for arguments
func (p *Parser) generateArgs(inputs []ABIInput, prefix string) (vars string, encode string) {
	var varLines []string
	var argNames []string

	for i, input := range inputs {
		varName := input.Name
		if varName == "" {
			varName = fmt.Sprintf("%sArg%d", prefix, i)
		}

		// Generate variable declaration based on type
		varType := p.solidityTypeToGo(input.Type)
		envVar := fmt.Sprintf("%s_%s", strings.ToUpper(prefix), strings.ToUpper(varName))
		
		varLines = append(varLines, fmt.Sprintf("%s %s = vm.envOr(\"%s\", %s);", 
			varType, varName, envVar, p.getDefaultValue(input.Type)))
		
		argNames = append(argNames, varName)
	}

	vars = strings.Join(varLines, "\n")
	encode = strings.Join(argNames, ", ")
	
	return vars, encode
}

// solidityTypeToGo converts Solidity types to script variable types
func (p *Parser) solidityTypeToGo(solType string) string {
	switch {
	case strings.HasPrefix(solType, "uint"):
		return "uint256"
	case strings.HasPrefix(solType, "int"):
		return "int256"
	case solType == "address":
		return "address"
	case solType == "bool":
		return "bool"
	case solType == "string":
		return "string"
	case strings.HasPrefix(solType, "bytes"):
		return "bytes"
	default:
		return solType
	}
}

// getDefaultValue returns a default value for a Solidity type
func (p *Parser) getDefaultValue(solType string) string {
	switch {
	case strings.HasPrefix(solType, "uint") || strings.HasPrefix(solType, "int"):
		return "0"
	case solType == "address":
		return "address(0)"
	case solType == "bool":
		return "false"
	case solType == "string":
		return `""`
	case strings.HasPrefix(solType, "bytes"):
		return `""`
	default:
		return "0"
	}
}

// findArtifact finds the artifact path for a contract
func (p *Parser) findArtifact(contractName string) string {
	// Clean contract name (remove .sol if present)
	contractName = strings.TrimSuffix(contractName, ".sol")
	
	// Common artifact locations
	patterns := []string{
		filepath.Join("out", contractName+".sol", contractName+".json"),
		filepath.Join("out", "**", contractName+".json"),
		filepath.Join("artifacts", "**", contractName+".json"),
	}

	for _, pattern := range patterns {
		fullPattern := filepath.Join(p.projectRoot, pattern)
		matches, _ := filepath.Glob(fullPattern)
		if len(matches) > 0 {
			return matches[0]
		}
	}

	// Try to find by walking the out directory
	outDir := filepath.Join(p.projectRoot, "out")
	var found string
	filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(path, contractName+".json") {
			found = path
			return filepath.SkipAll
		}
		return nil
	})

	return found
}