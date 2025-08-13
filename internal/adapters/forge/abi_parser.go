package forge

import (
	"context"
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ABIParserAdapter wraps the existing ABI parser to implement the ABIParser interface
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
			Inputs: convertParameters(contractABI.Constructor.Inputs),
		}
	}

	// Convert methods
	for _, method := range contractABI.Methods {
		result.Methods = append(result.Methods, domain.Method{
			Name:   method.Name,
			Inputs: convertParameters(method.Inputs),
		})
	}

	return result, nil
}

// FindInitializeMethod finds the initializer method in the ABI
func (a *ABIParserAdapter) FindInitializeMethod(abi *domain.ContractABI) *domain.Method {
	// Look for common initializer method names
	initializerNames := []string{"initialize", "init", "__init", "initializer"}
	
	for _, method := range abi.Methods {
		for _, name := range initializerNames {
			if strings.EqualFold(method.Name, name) {
				return &method
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
		Inputs: convertParametersToABI(contractABI.Constructor.Inputs),
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
		Inputs: convertParametersToABI(method.Inputs),
	}

	return a.parser.GenerateInitializerArgs(abiMethod)
}

// convertParameters converts ABI parameters to domain parameters
func convertParameters(inputs []abi.ABIInput) []domain.Parameter {
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

// convertParametersToABI converts domain parameters back to ABI parameters
func convertParametersToABI(params []domain.Parameter) []abi.ABIInput {
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