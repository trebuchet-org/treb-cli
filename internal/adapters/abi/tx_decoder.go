package abi

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// Well-known contract addresses
var (
	// CreateX factory address (deployed on multiple chains)
	CreateXAddress = common.HexToAddress("0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed")

	// Other well-known addresses
	MultiSendAddress    = common.HexToAddress("0x40A2aCCbd92BCA938b02010E17A5b8929b49130D") // Gnosis Safe MultiSend
	ProxyFactoryAddress = common.HexToAddress("0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67") // Gnosis Safe Proxy Factory
)

// TransactionDecoder provides utilities for decoding transaction data using ABI
type TransactionDecoder struct {
	abiResolver     usecase.ABIResolver
	deploymentsRepo usecase.DeploymentRepository
	log             *slog.Logger
	execution       *forge.HydratedRunResult
	label           map[common.Address]string
}

// NewTransactionDecoder creates a new transaction decoder
func NewTransactionDecoder(abiResolver usecase.ABIResolver, deploymentsRepo usecase.DeploymentRepository, execution *forge.HydratedRunResult, log *slog.Logger) *TransactionDecoder {
	decoder := &TransactionDecoder{
		abiResolver:     abiResolver,
		deploymentsRepo: deploymentsRepo,
		execution:       execution,
		log:             log.With("component", "TxDecoder"),
		label:           make(map[common.Address]string),
	}

	// Register well-known contracts
	decoder.label[CreateXAddress] = "CreateX"
	decoder.label[MultiSendAddress] = "MultiSend"
	decoder.label[ProxyFactoryAddress] = "SafeProxyFactory"

	return decoder
}

// DecodedTransaction represents a human-readable transaction
type DecodedTransaction struct {
	To           common.Address
	Label        string
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
	Value any
}

// DecodedOutput represents a decoded function output
type DecodedOutput struct {
	Name  string
	Type  string
	Value any
}

func (td *TransactionDecoder) GetArtifact(to common.Address) string {
	for _, deployment := range td.execution.Deployments {
		if deployment.Address == to {
			return deployment.Event.Artifact
		}
	}

	deployment, _ := td.deploymentsRepo.GetDeploymentByAddress(context.Background(), td.execution.ChainID, to.String())
	if deployment != nil {
		return fmt.Sprintf("%s:%s", deployment.Artifact.Path, deployment.ContractName)
	}

	return ""
}

func (td *TransactionDecoder) GetLabel(to common.Address) string {
	if label, exists := td.label[to]; exists {
		return label
	}

	artifact := td.GetArtifact(to)
	if artifact != "" && strings.Contains(artifact, ":") {
		parts := strings.Split(artifact, ":")
		if len(parts) > 1 {
			return parts[1]
		}
	}

	if artifact == "" {
		return to.String()
	}

	return artifact
}

func (td *TransactionDecoder) GetABI(to common.Address) *abi.ABI {
	abi, err := td.abiResolver.FindByAddress(context.Background(), to)
	if abi == nil {
		td.log.Debug("ABI Not Found", "address", to.String(), "err", err)
	}
	return abi
}

func (td *TransactionDecoder) DecodeTraceInfo(trace *forge.TraceInfo) *DecodedTransaction {
	// TODO: try to build *DecodedTransaction from trace.Decoded as well?
	to := trace.Address
	data, err := hex.DecodeString(strings.TrimPrefix(trace.Data, "0x"))
	if err != nil {
		td.log.Warn("Unable to decode raw data", "err", err, "data", trace.Data)
	}
	value, ok := big.NewInt(0).SetString(strings.TrimPrefix(trace.Value, "0x"), 16)
	if !ok {
		td.log.Warn("Unable to parse bigint", "value", trace.Value)
	}
	returnData, err := hex.DecodeString(strings.TrimPrefix(trace.Output, "0x"))
	if err != nil {
		td.log.Warn("Unable to parse output", "err", err, "output", trace.Output)
	}

	return td.DecodeTransaction(to, data, value, returnData)
}

// DecodeTransaction decodes transaction calldata and return data using registered ABIs
func (td *TransactionDecoder) DecodeTransaction(to common.Address, data []byte, value *big.Int, returnData []byte) *DecodedTransaction {
	decoded := &DecodedTransaction{
		To:           to,
		Label:        td.GetLabel(to),
		Value:        value,
		RawData:      hex.EncodeToString(data),
		IsDeployment: to == common.HexToAddress("0x0"),
	}

	// For deployments, we can't decode much
	if decoded.IsDeployment {
		decoded.Method = "constructor"
		return decoded
	}

	// Try to decode using registered ABI
	abi := td.GetABI(to)

	if abi == nil {
		decoded.Method = "unknown"
		return decoded
	}

	// Decode method call
	if len(data) >= 4 {
		method, err := abi.MethodById(data[:4])
		if err != nil {
			decoded.Method = "unknown"
			return decoded
		}
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

		// Decode return data if len(returnData) > 0 && len(method.Outputs) > 0 {
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

	if decoded.Method == "" {
		decoded.Method = "unknown"
	}

	return decoded
}

// FormatValue formats a decoded value for human display
func FormatValue(value any, valueType string) string {
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
		if dt.Label != "" {
			result += fmt.Sprintf(" %s%s%s", "\033[36m", dt.Label, "\033[0m")
		}
	} else {
		// Contract name or address
		if dt.Label != "" {
			result += fmt.Sprintf("%s%s%s.", "\033[36m", dt.Label, "\033[0m")
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
func (td *TransactionDecoder) DecodeConstructor(address common.Address, constructorArgs []byte) (*DecodedConstructor, error) {
	artifactLabel := td.GetLabel(address)
	abi := td.GetABI(address)

	if len(constructorArgs) == 0 {
		return &DecodedConstructor{
			Artifact: artifactLabel,
			Inputs:   []DecodedInput{},
			RawArgs:  constructorArgs,
		}, nil
	}

	if abi == nil {
		return &DecodedConstructor{
			Artifact: artifactLabel,
			Inputs:   []DecodedInput{},
			RawArgs:  constructorArgs,
		}, fmt.Errorf("ABI not found for artifact %s", artifactLabel)
	}

	// Get the constructor method
	if abi.Constructor.Inputs == nil {
		// No constructor defined, but args provided - this is unusual
		return &DecodedConstructor{
			Artifact: artifactLabel,
			Inputs:   []DecodedInput{},
			RawArgs:  constructorArgs,
		}, fmt.Errorf("no constructor found in ABI for %s", artifactLabel)
	}

	// Decode the arguments
	inputs, err := abi.Constructor.Inputs.Unpack(constructorArgs)
	if err != nil {
		return &DecodedConstructor{
			Artifact: artifactLabel,
			Inputs:   []DecodedInput{},
			RawArgs:  constructorArgs,
		}, fmt.Errorf("failed to decode constructor args: %w", err)
	}

	// Convert to DecodedInput format
	decodedInputs := make([]DecodedInput, 0, len(abi.Constructor.Inputs))
	for i, input := range abi.Constructor.Inputs {
		if i < len(inputs) {
			decodedInputs = append(decodedInputs, DecodedInput{
				Name:  input.Name,
				Type:  input.Type.String(),
				Value: inputs[i],
			})
		}
	}

	return &DecodedConstructor{
		Artifact: artifactLabel,
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
