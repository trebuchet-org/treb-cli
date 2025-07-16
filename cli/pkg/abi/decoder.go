package abi

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/bindings"
)

// Well-known contract addresses
var (
	// CreateX factory address (deployed on multiple chains)
	CreateXAddress = common.HexToAddress("0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed")

	// Other well-known addresses
	MultiSendAddress    = common.HexToAddress("0x40A2aCCbd92BCA938b02010E17A5b8929b49130D") // Gnosis Safe MultiSend
	ProxyFactoryAddress = common.HexToAddress("0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67") // Gnosis Safe Proxy Factory
)

// ABIResolver is an interface for resolving ABIs for contracts
type ABIResolver interface {
	// ResolveByAddress attempts to find and load the ABI for a given address
	// Returns the contract name and ABI JSON, or empty strings if not found
	ResolveByAddress(address common.Address) (contractName string, abiJSON string, isProxy bool, implAddress *common.Address)

	// ResolveByArtifact attempts to find and load the ABI for a given artifact name
	// Returns the contract name and ABI JSON, or empty strings if not found
	ResolveByArtifact(artifact string) (contractName string, abiJSON string)
}

// TransactionDecoder provides utilities for decoding transaction data using ABI
type TransactionDecoder struct {
	contractABIs map[common.Address]*abi.ABI
	artifactMap  map[common.Address]string
	// Proxy relationships: proxy address -> implementation address
	proxyImplementations map[common.Address]common.Address
	// Optional ABI resolver for on-demand loading
	abiResolver ABIResolver
}

// NewTransactionDecoder creates a new transaction decoder
func NewTransactionDecoder() *TransactionDecoder {
	decoder := &TransactionDecoder{
		contractABIs:         make(map[common.Address]*abi.ABI),
		artifactMap:          make(map[common.Address]string),
		proxyImplementations: make(map[common.Address]common.Address),
	}

	// Register well-known contracts
	decoder.artifactMap[CreateXAddress] = "CreateX"
	decoder.artifactMap[MultiSendAddress] = "MultiSend"
	decoder.artifactMap[ProxyFactoryAddress] = "SafeProxyFactory"

	_ = decoder.RegisterContract(CreateXAddress, "CreateX", bindings.CreateXMetaData.ABI)

	return decoder
}

// SetABIResolver sets the ABI resolver for on-demand loading
func (td *TransactionDecoder) SetABIResolver(resolver ABIResolver) {
	td.abiResolver = resolver
}

// RegisterContract registers a contract's ABI for transaction decoding
func (td *TransactionDecoder) RegisterContract(address common.Address, artifact string, abiJSON string) error {
	contractABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return fmt.Errorf("failed to parse ABI for %s: %w", artifact, err)
	}

	td.contractABIs[address] = &contractABI
	td.artifactMap[address] = artifact
	return nil
}

// RegisterContractByArtifact registers a contract's ABI by artifact name only (no address)
// This is useful for decoding constructor arguments before deployment
func (td *TransactionDecoder) RegisterContractByArtifact(artifact string, abiJSON string) error {
	contractABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return fmt.Errorf("failed to parse ABI for %s: %w", artifact, err)
	}

	// Use a special zero address for artifact-only registrations
	artifactKey := common.HexToAddress("0x" + strings.Repeat("0", 39) + "1")
	// Hash the artifact name to create a unique address
	hash := crypto.Keccak256([]byte(artifact))
	copy(artifactKey[:], hash[:20])

	td.contractABIs[artifactKey] = &contractABI
	td.artifactMap[artifactKey] = artifact
	return nil
}

// RegisterProxyRelationship registers a proxy -> implementation relationship
func (td *TransactionDecoder) RegisterProxyRelationship(proxyAddress, implementationAddress common.Address) {
	td.proxyImplementations[proxyAddress] = implementationAddress
}

// DecodedTransaction represents a human-readable transaction
type DecodedTransaction struct {
	To           common.Address
	ToArtifact   string
	Method       string
	Inputs       []DecodedInput
	Value        *big.Int
	ReturnData   []DecodedOutput
	IsDeployment bool
	RawData      string
}

// DecodedInput represents a decoded function input
type DecodedInput struct {
	Name  string
	Type  string
	Value interface{}
}

// DecodedOutput represents a decoded function output
type DecodedOutput struct {
	Name  string
	Type  string
	Value interface{}
}

// DecodeTransaction decodes transaction calldata and return data using registered ABIs
func (td *TransactionDecoder) DecodeTransaction(to common.Address, data []byte, value *big.Int, returnData []byte) *DecodedTransaction {
	decoded := &DecodedTransaction{
		To:           to,
		Value:        value,
		RawData:      hex.EncodeToString(data),
		IsDeployment: to == common.HexToAddress("0x0"),
	}

	// Check if we have the artifact name
	if artifact, exists := td.artifactMap[to]; exists {
		decoded.ToArtifact = artifact
	}

	// For deployments, we can't decode much
	if decoded.IsDeployment {
		decoded.Method = "constructor"
		return decoded
	}

	// Try to decode using registered ABI
	contractABI, exists := td.contractABIs[to]

	// If this is a proxy and we don't have the ABI, check if we have the implementation's ABI
	if !exists {
		if implAddr, isProxy := td.proxyImplementations[to]; isProxy {
			contractABI, exists = td.contractABIs[implAddr]
			// The artifact name should already include proxy indicator from registration
		}
	}

	// If still not found, try the ABI resolver
	if !exists && td.abiResolver != nil {
		if contractName, abiJSON, isProxy, implAddr := td.abiResolver.ResolveByAddress(to); abiJSON != "" {
			// Register the contract
			if err := td.RegisterContract(to, contractName, abiJSON); err == nil {
				contractABI = td.contractABIs[to]
				exists = true

				// Update the artifact name with the resolved name
				td.artifactMap[to] = contractName
				decoded.ToArtifact = contractName // Update the decoded transaction's artifact name

				// Handle proxy relationship if discovered
				if isProxy && implAddr != nil {
					td.RegisterProxyRelationship(to, *implAddr)
				}
			}
		}
	}

	if !exists {
		decoded.Method = "unknown"
		return decoded
	}

	// Decode method call
	if len(data) >= 4 {
		methodID := data[:4]

		// Find matching method
		for _, method := range contractABI.Methods {
			if bytes.Equal(method.ID[:4], methodID) {
				decoded.Method = method.RawName

				// Decode inputs
				if len(data) > 4 {
					inputs, err := method.Inputs.Unpack(data[4:])
					if err == nil {
						for i, input := range method.Inputs {
							if i < len(inputs) {
								decoded.Inputs = append(decoded.Inputs, DecodedInput{
									Name:  input.Name,
									Type:  input.Type.String(),
									Value: inputs[i],
								})
							}
						}
					}
				}

				// Decode return data
				if len(returnData) > 0 && len(method.Outputs) > 0 {
					outputs, err := method.Outputs.Unpack(returnData)
					if err == nil {
						for i, output := range method.Outputs {
							if i < len(outputs) {
								decoded.ReturnData = append(decoded.ReturnData, DecodedOutput{
									Name:  output.Name,
									Type:  output.Type.String(),
									Value: outputs[i],
								})
							}
						}
					}
				}
				break
			}
		}
	}

	if decoded.Method == "" {
		decoded.Method = "unknown"
	}

	return decoded
}

// FormatValue formats a decoded value for human display
func FormatValue(value interface{}, valueType string) string {
	switch v := value.(type) {
	case common.Address:
		return v.Hex()
	case *big.Int:
		if strings.Contains(valueType, "uint") {
			return v.String()
		}
		return v.String()
	case []byte:
		if len(v) == 0 {
			return "0x"
		}
		if len(v) <= 32 {
			return hexutil.Encode(v)
		}
		// Truncate long byte arrays
		return fmt.Sprintf("%s...(%d bytes)", hexutil.Encode(v[:16]), len(v))
	case string:
		if len(v) > 50 {
			return fmt.Sprintf("%.50s...(%d chars)", v, len(v))
		}
		return fmt.Sprintf(`"%s"`, v)
	case bool:
		return fmt.Sprintf("%t", v)
	case [32]byte:
		return hexutil.Encode(v[:])
	default:
		// Try JSON marshaling for complex types
		if jsonBytes, err := json.Marshal(v); err == nil {
			jsonStr := string(jsonBytes)
			if len(jsonStr) > 100 {
				return fmt.Sprintf("%.100s...(%d chars)", jsonStr, len(jsonStr))
			}
			return jsonStr
		}
		return fmt.Sprintf("%v", v)
	}
}

// FormatDecodedTransaction formats a decoded transaction for display
func (dt *DecodedTransaction) Format() string {
	var parts []string

	// Target info
	if dt.ToArtifact != "" {
		parts = append(parts, fmt.Sprintf("To: %s (%s)", dt.ToArtifact, dt.To.Hex()[:10]+"..."))
	} else {
		parts = append(parts, fmt.Sprintf("To: %s", dt.To.Hex()[:10]+"..."))
	}

	// Method info
	if dt.IsDeployment {
		parts = append(parts, "Method: constructor")
	} else {
		parts = append(parts, fmt.Sprintf("Method: %s", dt.Method))
	}

	// Value
	if dt.Value != nil && dt.Value.Cmp(big.NewInt(0)) > 0 {
		parts = append(parts, fmt.Sprintf("Value: %s ETH", dt.Value.String()))
	}

	// Inputs
	if len(dt.Inputs) > 0 {
		inputStrs := make([]string, 0, len(dt.Inputs))
		for _, input := range dt.Inputs {
			if input.Name != "" {
				inputStrs = append(inputStrs, fmt.Sprintf("%s: %s", input.Name, FormatValue(input.Value, input.Type)))
			} else {
				inputStrs = append(inputStrs, FormatValue(input.Value, input.Type))
			}
		}
		if len(inputStrs) <= 3 {
			parts = append(parts, fmt.Sprintf("Inputs: %s", strings.Join(inputStrs, ", ")))
		} else {
			parts = append(parts, fmt.Sprintf("Inputs: %s, ...(%d total)", strings.Join(inputStrs[:3], ", "), len(inputStrs)))
		}
	}

	// Return data
	if len(dt.ReturnData) > 0 {
		outputStrs := make([]string, 0, len(dt.ReturnData))
		for _, output := range dt.ReturnData {
			if output.Name != "" {
				outputStrs = append(outputStrs, fmt.Sprintf("%s: %s", output.Name, FormatValue(output.Value, output.Type)))
			} else {
				outputStrs = append(outputStrs, FormatValue(output.Value, output.Type))
			}
		}
		if len(outputStrs) <= 2 {
			parts = append(parts, fmt.Sprintf("Returns: %s", strings.Join(outputStrs, ", ")))
		} else {
			parts = append(parts, fmt.Sprintf("Returns: %s, ...(%d total)", strings.Join(outputStrs[:2], ", "), len(outputStrs)))
		}
	}

	return strings.Join(parts, " | ")
}

// FormatCompact formats a decoded transaction in a compact, user-friendly way: sender -> Contract.method(args)
func (dt *DecodedTransaction) FormatCompact(sender common.Address) string {
	return dt.FormatCompactWithReconciler(sender, nil)
}

// FormatCompactWithReconciler formats a decoded transaction with address reconciliation
func (dt *DecodedTransaction) FormatCompactWithReconciler(sender common.Address, reconciler func(common.Address) string) string {
	// Start with sender
	senderStr := sender.Hex()[:10] + "..."
	if reconciler != nil {
		senderStr = reconciler(sender)
	}
	result := fmt.Sprintf("%s%s%s -> ", "\033[36m", senderStr, "\033[0m")

	// Add target
	if dt.IsDeployment {
		result += fmt.Sprintf("%sDeploy%s", "\033[32m", "\033[0m")
		if dt.ToArtifact != "" {
			result += fmt.Sprintf(" %s%s%s", "\033[36m", dt.ToArtifact, "\033[0m")
		}
	} else {
		// Contract name or address
		if dt.ToArtifact != "" {
			result += fmt.Sprintf("%s%s%s.", "\033[36m", dt.ToArtifact, "\033[0m")
		} else {
			result += fmt.Sprintf("%s%s%s.", "\033[90m", dt.To.Hex()[:10]+"...", "\033[0m")
		}

		// Method name
		if dt.Method != "unknown" {
			result += fmt.Sprintf("%s%s%s", "\033[33m", dt.Method, "\033[0m")
		} else {
			result += fmt.Sprintf("%s%s%s", "\033[90m", "unknown", "\033[0m")
		}

		// Arguments
		if len(dt.Inputs) > 0 {
			argStrs := make([]string, 0, len(dt.Inputs))
			for _, input := range dt.Inputs {
				// Format value compactly
				val := FormatValue(input.Value, input.Type)
				// Shorten long values
				if len(val) > 40 {
					val = val[:37] + "..."
				}
				argStrs = append(argStrs, val)
			}

			// Always show all arguments
			result += fmt.Sprintf("(%s)", strings.Join(argStrs, ", "))
		} else {
			result += "()"
		}
	}

	// Add value if non-zero
	if dt.Value != nil && dt.Value.Cmp(big.NewInt(0)) > 0 {
		// Convert to ether (assuming value is in wei)
		ethValue := new(big.Float).SetInt(dt.Value)
		ethValue.Quo(ethValue, big.NewFloat(1e18))
		result += fmt.Sprintf(" %s{value: %s ETH}%s", "\033[90m", ethValue.Text('f', 6), "\033[0m")
	}

	return result
}

// DecodedConstructor represents decoded constructor arguments
type DecodedConstructor struct {
	Artifact string
	Inputs   []DecodedInput
	RawArgs  []byte
}

// DecodeConstructor decodes constructor arguments for a contract
func (td *TransactionDecoder) DecodeConstructor(artifact string, constructorArgs []byte) (*DecodedConstructor, error) {
	if len(constructorArgs) == 0 {
		return &DecodedConstructor{
			Artifact: artifact,
			Inputs:   []DecodedInput{},
			RawArgs:  constructorArgs,
		}, nil
	}

	// Try to find the ABI for this artifact
	var contractABI *abi.ABI
	var contractName string

	// Extract contract name from artifact path (e.g., "src/Counter.sol:Counter" -> "Counter")
	if idx := strings.LastIndex(artifact, ":"); idx != -1 {
		contractName = artifact[idx+1:]
	} else {
		contractName = artifact
	}

	// First check if we have it in our registered ABIs by artifact name
	for addr, artifactName := range td.artifactMap {
		if artifactName == artifact || artifactName == contractName {
			if abi, exists := td.contractABIs[addr]; exists {
				contractABI = abi
				break
			}
		}
	}

	// If not found and we have an ABI resolver, try to resolve by artifact
	if contractABI == nil && td.abiResolver != nil {
		if _, abiJSON := td.abiResolver.ResolveByArtifact(artifact); abiJSON != "" {
			// Parse and store the ABI
			parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
			if err == nil {
				contractABI = &parsedABI
				// Also register it for future use
				_ = td.RegisterContractByArtifact(artifact, abiJSON)
			}
		}
	}

	if contractABI == nil {
		return &DecodedConstructor{
			Artifact: artifact,
			Inputs:   []DecodedInput{},
			RawArgs:  constructorArgs,
		}, fmt.Errorf("ABI not found for artifact %s", artifact)
	}

	// Get the constructor method
	if contractABI.Constructor.Inputs == nil {
		// No constructor defined, but args provided - this is unusual
		return &DecodedConstructor{
			Artifact: artifact,
			Inputs:   []DecodedInput{},
			RawArgs:  constructorArgs,
		}, fmt.Errorf("no constructor found in ABI for %s", artifact)
	}

	// Decode the arguments
	inputs, err := contractABI.Constructor.Inputs.Unpack(constructorArgs)
	if err != nil {
		return &DecodedConstructor{
			Artifact: artifact,
			Inputs:   []DecodedInput{},
			RawArgs:  constructorArgs,
		}, fmt.Errorf("failed to decode constructor args: %w", err)
	}

	// Convert to DecodedInput format
	decodedInputs := make([]DecodedInput, 0, len(contractABI.Constructor.Inputs))
	for i, input := range contractABI.Constructor.Inputs {
		if i < len(inputs) {
			decodedInputs = append(decodedInputs, DecodedInput{
				Name:  input.Name,
				Type:  input.Type.String(),
				Value: inputs[i],
			})
		}
	}

	return &DecodedConstructor{
		Artifact: artifact,
		Inputs:   decodedInputs,
		RawArgs:  constructorArgs,
	}, nil
}

// FormatConstructorArgs formats decoded constructor arguments for display
func (dc *DecodedConstructor) FormatCompact() string {
	if len(dc.Inputs) == 0 {
		return ""
	}

	argStrs := make([]string, 0, len(dc.Inputs))
	for _, input := range dc.Inputs {
		val := FormatValue(input.Value, input.Type)
		// Shorten long values
		if len(val) > 40 {
			val = val[:37] + "..."
		}
		// Include parameter name if available
		if input.Name != "" {
			argStrs = append(argStrs, fmt.Sprintf("%s: %s", input.Name, val))
		} else {
			argStrs = append(argStrs, val)
		}
	}

	return fmt.Sprintf("(%s)", strings.Join(argStrs, ", "))
}
