package abi

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
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
		Methods:        []Method{},
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

// DecodeConstructorArgs decodes constructor arguments for a given contract
func (p *Parser) DecodeConstructorArgs(contractName string, constructorArgs []byte) (string, error) {
	if len(constructorArgs) == 0 {
		return "", nil
	}

	// Try common patterns first
	decodedArgs := p.tryCommonPatterns(contractName, constructorArgs)
	if decodedArgs != "" {
		return decodedArgs, nil
	}

	// Try to decode using the contract's ABI
	contractABI, err := p.ParseContractABI(contractName)
	if err != nil {
		return "", fmt.Errorf("failed to parse contract ABI: %w", err)
	}

	if !contractABI.HasConstructor || len(contractABI.Constructor.Inputs) == 0 {
		return "", fmt.Errorf("contract has no constructor or no inputs")
	}

	// Convert our ABI inputs to go-ethereum ABI arguments
	arguments := make(abi.Arguments, len(contractABI.Constructor.Inputs))
	for i, input := range contractABI.Constructor.Inputs {
		abiType, err := abi.NewType(input.Type, input.InternalType, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create ABI type for %s: %w", input.Type, err)
		}
		arguments[i] = abi.Argument{
			Name: input.Name,
			Type: abiType,
		}
	}

	// Unpack the arguments
	values, err := arguments.Unpack(constructorArgs)
	if err != nil {
		return "", fmt.Errorf("failed to unpack constructor arguments: %w", err)
	}

	// Format the decoded arguments
	return p.formatDecodedArgs(contractABI.Constructor.Inputs, values), nil
}

// tryCommonPatterns tries to decode constructor arguments using common patterns
func (p *Parser) tryCommonPatterns(contractName string, args []byte) string {
	lowerName := strings.ToLower(contractName)

	// ERC20 Token pattern: (string name, string symbol, uint256 totalSupply)
	if (strings.Contains(lowerName, "token") || strings.Contains(lowerName, "erc20")) && len(args) == 224 {
		stringType, _ := abi.NewType("string", "", nil)
		uint256Type, _ := abi.NewType("uint256", "", nil)

		arguments := abi.Arguments{
			{Type: stringType, Name: "name"},
			{Type: stringType, Name: "symbol"},
			{Type: uint256Type, Name: "totalSupply"},
		}

		values, err := arguments.Unpack(args)
		if err == nil && len(values) == 3 {
			if name, ok := values[0].(string); ok {
				if symbol, ok := values[1].(string); ok {
					if totalSupply, ok := values[2].(*big.Int); ok {
						return fmt.Sprintf(`"%s", "%s", %s`, name, symbol, FormatTokenAmount(totalSupply))
					}
				}
			}
		}
	}

	// ERC20 Token with decimals pattern: (string name, string symbol, uint8 decimals, uint256 totalSupply)
	if (strings.Contains(lowerName, "token") || strings.Contains(lowerName, "erc20")) && len(args) >= 256 {
		stringType, _ := abi.NewType("string", "", nil)
		uint8Type, _ := abi.NewType("uint8", "", nil)
		uint256Type, _ := abi.NewType("uint256", "", nil)

		arguments := abi.Arguments{
			{Type: stringType, Name: "name"},
			{Type: stringType, Name: "symbol"},
			{Type: uint8Type, Name: "decimals"},
			{Type: uint256Type, Name: "totalSupply"},
		}

		values, err := arguments.Unpack(args)
		if err == nil && len(values) == 4 {
			if name, ok := values[0].(string); ok {
				if symbol, ok := values[1].(string); ok {
					if decimals, ok := values[2].(uint8); ok {
						if totalSupply, ok := values[3].(*big.Int); ok {
							return fmt.Sprintf(`"%s", "%s", %d, %s`, name, symbol, decimals, FormatTokenAmount(totalSupply))
						}
					}
				}
			}
		}
	}

	// Simple address pattern (owner, admin, etc.)
	if len(args) == 32 {
		addressType, _ := abi.NewType("address", "", nil)
		arguments := abi.Arguments{{Type: addressType, Name: "address"}}

		values, err := arguments.Unpack(args)
		if err == nil && len(values) == 1 {
			if addr, ok := values[0].(common.Address); ok {
				return addr.Hex()
			}
		}
	}

	// Add more patterns as needed...

	return ""
}

// formatDecodedArgs formats decoded arguments into a readable string
func (p *Parser) formatDecodedArgs(inputs []ABIInput, values []interface{}) string {
	if len(inputs) != len(values) {
		return ""
	}

	var formatted []string
	for i, input := range inputs {
		formattedValue := p.formatValue(input.Type, values[i])
		formatted = append(formatted, formattedValue)
	}

	return strings.Join(formatted, ", ")
}

// formatValue formats a single value based on its type
func (p *Parser) formatValue(solidityType string, value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, v)
	case *big.Int:
		if strings.Contains(solidityType, "uint") && strings.Contains(solidityType, "256") {
			// Check if it might be a token amount (has many zeros)
			if v.BitLen() > 60 { // Likely a token amount
				return FormatTokenAmount(v)
			}
		}
		return v.String()
	case bool:
		return fmt.Sprintf("%t", v)
	case common.Address:
		return v.Hex()
	case []byte:
		if len(v) <= 32 {
			return fmt.Sprintf("0x%x", v)
		}
		return fmt.Sprintf("0x%x... (%d bytes)", v[:16], len(v))
	default:
		// Handle arrays
		if slice, ok := value.([]interface{}); ok {
			var elements []string
			for _, elem := range slice {
				// Recursively format array elements
				elements = append(elements, p.formatValue(strings.TrimSuffix(solidityType, "[]"), elem))
			}
			return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
		}
		return fmt.Sprintf("%v", value)
	}
}

// FormatTokenAmount formats a big.Int as a human-readable token amount
func FormatTokenAmount(amount *big.Int) string {
	if amount == nil {
		return "0"
	}

	// Common token decimals
	decimals := []struct {
		name     string
		value    *big.Int
		exponent int
	}{
		{"18", new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), 18},
		{"6", new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil), 6},
		{"8", new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil), 8},
	}

	// Try each decimal to see if we get a clean division
	for _, d := range decimals {
		whole := new(big.Int).Div(amount, d.value)
		remainder := new(big.Int).Mod(amount, d.value)

		if remainder.Cmp(big.NewInt(0)) == 0 && whole.Sign() > 0 {
			// Clean division - format nicely
			wholeStr := whole.String()

			// Add comma separators for readability
			if len(wholeStr) > 3 {
				wholeStr = addCommas(wholeStr)
			}

			return fmt.Sprintf("%s * 10^%d", wholeStr, d.exponent)
		}
	}

	// Not a clean token amount, just return the raw value with commas
	return addCommas(amount.String())
}

// addCommas adds comma separators to a number string
func addCommas(s string) string {
	if len(s) <= 3 {
		return s
	}

	// Work backwards adding commas
	var result []byte
	for i := len(s) - 1; i >= 0; i-- {
		if (len(s)-i-1)%3 == 0 && i != len(s)-1 {
			result = append([]byte{','}, result...)
		}
		result = append([]byte{s[i]}, result...)
	}

	return string(result)
}
