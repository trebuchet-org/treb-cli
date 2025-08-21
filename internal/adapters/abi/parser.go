package abi

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/trebuchet-org/treb-cli/internal/domain/bindings"
)

// Parser handles ABI parsing from Foundry artifacts
type Parser struct {
	trebContract    *bindings.Treb
	createXContract *bindings.CreateX
}

// NewParser creates a new ABI parser
func NewParser(projectRoot string) *Parser {
	return &Parser{
		trebContract:    bindings.NewTreb(),
		createXContract: bindings.NewCreateX(),
	}
}

// FindInitializeMethod finds an initializer method in the ABI
func (p *Parser) FindInitializeMethod(abi *abi.ABI) *abi.Method {
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
func (p *Parser) GenerateConstructorArgs(abi *abi.ABI) (vars string, encode string) {
	if abi != nil && len(abi.Constructor.Inputs) == 0 {
		return "", ""
	}

	return p.generateArgs(abi.Constructor.Inputs, "constructor")
}

// GenerateInitializerArgs generates variable declarations and encoding for initializer arguments
func (p *Parser) GenerateInitializerArgs(method *abi.Method) (vars, encode, sig string) {
	if method == nil || len(method.Inputs) == 0 {
		return "", "", ""
	}
	vars, encode = p.generateArgs(method.Inputs, "init")
	return vars, encode, method.Sig
}

// generateArgs generates variable declarations and encoding for arguments
func (p *Parser) generateArgs(inputs abi.Arguments, prefix string) (vars string, encode string) {
	var varLines []string
	var argNames []string

	for i, input := range inputs {
		varName := input.Name
		if varName == "" {
			varName = fmt.Sprintf("%sArg%d", prefix, i)
		}

		// Generate variable declaration based on type
		varType := p.solidityTypeToGo(input.Type.String())
		
		// Add memory keyword for reference types
		memoryKeyword := ""
		if varType == "string" || varType == "bytes" || strings.Contains(varType, "[]") {
			memoryKeyword = " memory"
		}

		varLines = append(varLines, fmt.Sprintf("        %s%s %s = %s;",
			varType, memoryKeyword, varName, p.getDefaultValue(input.Type.String())))

		argNames = append(argNames, varName)
	}

	vars = strings.Join(varLines, "\n")
	encode = fmt.Sprintf("return abi.encode(%s);", strings.Join(argNames, ", "))

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
