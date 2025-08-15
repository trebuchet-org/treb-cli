package forge

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/adapters/abi"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ABIParserAdapter wraps the internal ABI parser to implement the ABIParser interface
type ABIParserAdapter struct {
	parser *abi.Parser
}

// NewABIParserAdapter creates a new ABI parser adapter
func NewABIParserAdapter(cfg *config.RuntimeConfig) (*ABIParserAdapter, error) {
	parser := abi.NewParser(cfg.ProjectRoot)
	return &ABIParserAdapter{parser: parser}, nil
}

// ParseContractABI parses a contract's ABI
func (a *ABIParserAdapter) ParseContractABI(ctx context.Context, contractName string) (*domain.ContractABI, error) {
	contractABI, err := a.parser.ParseContractABI(contractName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI for %s: %w", contractName, err)
	}

	// Convert from abi.ContractABI to domain.ContractABI
	result := &domain.ContractABI{
		Name:           contractName,
		HasConstructor: contractABI.HasConstructor,
		Methods:        make([]domain.Method, 0, len(contractABI.Methods)),
	}

	// Convert constructor
	if contractABI.Constructor != nil {
		result.Constructor = &domain.Constructor{
			Inputs: convertABIInputsToParameters(contractABI.Constructor.Inputs),
		}
	}

	// Convert methods
	for _, method := range contractABI.Methods {
		result.Methods = append(result.Methods, domain.Method{
			Name:   method.Name,
			Inputs: convertABIInputsToParameters(method.Inputs),
		})
	}

	return result, nil
}

// FindInitializeMethod finds the initializer method in the ABI
func (a *ABIParserAdapter) FindInitializeMethod(contractABI *domain.ContractABI) *domain.Method {
	// Convert to internal ABI type
	abiContract := &abi.ContractABI{
		Methods: make([]abi.Method, 0, len(contractABI.Methods)),
	}
	
	for _, method := range contractABI.Methods {
		abiContract.Methods = append(abiContract.Methods, abi.Method{
			Name:   method.Name,
			Inputs: convertParametersToABIInputs(method.Inputs),
		})
	}
	
	// Use internal parser's method
	if initMethod := a.parser.FindInitializeMethod(abiContract); initMethod != nil {
		// Find corresponding domain method
		for i, method := range contractABI.Methods {
			if method.Name == initMethod.Name {
				return &contractABI.Methods[i]
			}
		}
	}
	
	return nil
}

// GenerateConstructorArgs generates constructor argument handling code
func (a *ABIParserAdapter) GenerateConstructorArgs(contractABI *domain.ContractABI) (vars string, encode string) {
	// Convert back to abi types for generation
	if contractABI.Constructor == nil || len(contractABI.Constructor.Inputs) == 0 {
		return "", ""
	}

	abiConstructor := &abi.ABIConstructor{
		Type:   "constructor",
		Inputs: convertParametersToABIInputs(contractABI.Constructor.Inputs),
	}

	abiContractABI := &abi.ContractABI{
		Constructor:    abiConstructor,
		HasConstructor: true,
	}

	return a.parser.GenerateConstructorArgs(abiContractABI)
}

// GenerateInitializerArgs generates initializer argument handling code
func (a *ABIParserAdapter) GenerateInitializerArgs(method *domain.Method) (vars string, encode string) {
	if method == nil || len(method.Inputs) == 0 {
		return "", ""
	}

	// Convert to abi.Method for generation
	abiMethod := &abi.Method{
		Name:   method.Name,
		Type:   "function",
		Inputs: convertParametersToABIInputs(method.Inputs),
	}

	return a.parser.GenerateInitializerArgs(abiMethod)
}

// convertABIInputsToParameters converts ABI parameters to domain parameters
func convertABIInputsToParameters(inputs []abi.ABIInput) []domain.Parameter {
	params := make([]domain.Parameter, 0, len(inputs))
	for _, input := range inputs {
		params = append(params, domain.Parameter{
			Name:         input.Name,
			Type:         input.Type,
			InternalType: input.InternalType,
		})
	}
	return params
}

// convertParametersToABIInputs converts domain parameters back to ABI parameters
func convertParametersToABIInputs(params []domain.Parameter) []abi.ABIInput {
	inputs := make([]abi.ABIInput, 0, len(params))
	for _, param := range params {
		inputs = append(inputs, abi.ABIInput{
			Name:         param.Name,
			Type:         param.Type,
			InternalType: param.InternalType,
		})
	}
	return inputs
}

// Ensure the adapter implements the interface
var _ usecase.ABIParser = (*ABIParserAdapter)(nil)