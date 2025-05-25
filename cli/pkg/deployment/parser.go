package deployment

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// Parser handles parsing of deployment outputs
type Parser struct{}

// NewParser creates a new deployment parser
func NewParser() *Parser {
	return &Parser{}
}

type DeploymentOutput struct {
	Address               string `json:"address"`
	PredictedAddress      string `json:"predicted_address"`
	Salt                  string `json:"salt"`
	InitCodeHash          string `json:"init_code_hash"`
	Status                string `json:"status"`
	DeploymentType        string `json:"deployment_type"`
	Strategy              string `json:"strategy"`
	BlockNumber           string `json:"block_number"`
	ConstructorArgs       string `json:"constructor_args"`
	SafeTxHash            string `json:"safe_tx_hash"`
	ImplementationAddress string `json:"implementation_address"`
	LibraryAddress        string `json:"library_address"`
	ProxyInitializer      string `json:"proxy_initializer"`
}

// ParseDeploymentResult parses deployment results from script output
func (p *Parser) ParseDeploymentResult(output string) (DeploymentOutput, error) {
	result := DeploymentOutput{}

	// Parse structured output between === DEPLOYMENT_RESULT === and === END_DEPLOYMENT ===
	startPattern := "  === DEPLOYMENT_RESULT ===\n"
	endPattern := "  === END_DEPLOYMENT ===\n"

	startIdx := strings.Index(output, startPattern)
	endIdx := strings.Index(output, endPattern)

	// Extract the section between markers
	deploymentSection := output[startIdx+len(startPattern) : endIdx]

	// Parse key:value pairs
	lines := strings.Split(deploymentSection, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split on first colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Map to expected field names
			switch key {
			case "ADDRESS":
				result.Address = value
			case "PREDICTED":
				result.PredictedAddress = value
			case "SALT":
				result.Salt = value
			case "INIT_CODE_HASH":
				result.InitCodeHash = value
			case "STATUS":
				result.Status = value
			case "DEPLOYMENT_TYPE":
				result.DeploymentType = value
			case "STRATEGY":
				result.Strategy = value
			case "BLOCK_NUMBER":
				result.BlockNumber = value
			case "CONSTRUCTOR_ARGS":
				result.ConstructorArgs = value
			case "SAFE_TX_HASH":
				result.SafeTxHash = value
			case "IMPLEMENTATION_ADDRESS":
				result.ImplementationAddress = value
			case "LIBRARY_ADDRESS":
				result.LibraryAddress = value
			case "PROXY_INITIALIZER":
				result.ProxyInitializer = value
			}
		}
	}

	return result, nil
}

// ParsePredictionOutput parses prediction output from script
func (p *Parser) ParsePredictionOutput(output string) (*types.PredictResult, error) {
	// For predictions, use the same structured parser since they use the same format
	parsed, err := p.ParseDeploymentResult(output)
	if err != nil {
		return nil, err
	}

	result := &types.PredictResult{}

	// Get predicted address - use ADDRESS or PREDICTED field
	if addr := parsed.Address; addr != "" {
		result.Address = common.HexToAddress(addr)
	} else if addr := parsed.PredictedAddress; addr != "" {
		result.Address = common.HexToAddress(addr)
	}

	// Get salt
	if salt := parsed.Salt; salt != "" {
		saltBytes, _ := hex.DecodeString(strings.TrimPrefix(salt, "0x"))
		copy(result.Salt[:], saltBytes)
	}

	return result, nil
}

// ParseLibraryAddress parses library address from deployment output
func (p *Parser) ParseLibraryAddress(output string) (common.Address, error) {
	// Use structured parser to get library address
	parsed, err := p.ParseDeploymentResult(output)
	if err != nil {
		return common.Address{}, err
	}

	// Check for library address
	if addr := parsed.LibraryAddress; addr != "" {
		return common.HexToAddress(addr), nil
	}

	return common.Address{}, fmt.Errorf("library address not found in output")
}

// HandleForgeError extracts meaningful error messages from forge output
func (p *Parser) HandleForgeError(err error, output []byte) error {
	outputStr := string(output)

	// Check for common error patterns
	if strings.Contains(outputStr, "DeploymentAlreadyExists") {
		re := regexp.MustCompile(`Contract already deployed at: (0x[a-fA-F0-9]{40})`)
		if match := re.FindStringSubmatch(outputStr); len(match) > 1 {
			return fmt.Errorf("contract already deployed at %s", match[1])
		}
		return fmt.Errorf("contract already deployed")
	}

	if strings.Contains(outputStr, "insufficient funds") {
		return fmt.Errorf("insufficient funds for deployment")
	}

	if strings.Contains(outputStr, "nonce too low") {
		return fmt.Errorf("nonce too low - transaction may have already been sent")
	}

	if strings.Contains(outputStr, "replacement transaction underpriced") {
		return fmt.Errorf("replacement transaction underpriced - increase gas price")
	}

	// Check for CreateX collision error
	if strings.Contains(outputStr, "CreateCollision") || strings.Contains(outputStr, "create collision") {
		// Try to extract the address that already exists
		addrRegex := regexp.MustCompile(`address\s+(0x[a-fA-F0-9]{40})`)
		var existingAddr string
		if match := addrRegex.FindStringSubmatch(outputStr); len(match) > 1 {
			existingAddr = match[1]
		}

		// More helpful message about deployments.json
		if existingAddr != "" {
			return fmt.Errorf("contract already exists at %s (CreateX collision). This was most likely deployed but not found in the current deployments.json - make sure you have the latest deployments.json. Alternatively, use a different label with --label flag", existingAddr)
		}
		return fmt.Errorf("contract already exists at this address (CreateX collision). This was most likely deployed but not found in the current deployments.json - make sure you have the latest deployments.json. Alternatively, use a different label with --label flag")
	}

	// Extract revert reason
	revertRegex := regexp.MustCompile(`reverted with reason string '([^']+)'`)
	if match := revertRegex.FindStringSubmatch(outputStr); len(match) > 1 {
		return fmt.Errorf("transaction reverted: %s", match[1])
	}

	// Extract any explicit error message
	errorRegex := regexp.MustCompile(`Error:\s+(.+)`)
	if match := errorRegex.FindStringSubmatch(outputStr); len(match) > 1 {
		return fmt.Errorf(match[1])
	}

	// If no specific error found, return the original error with partial output
	if len(output) > 500 {
		return fmt.Errorf("%v\nOutput (last 500 chars): ...%s", err, outputStr[len(outputStr)-500:])
	}
	return fmt.Errorf("%v\nOutput: %s", err, outputStr)
}
