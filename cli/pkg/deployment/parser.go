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

// ParseDeploymentResult parses deployment results from script output
func (p *Parser) ParseDeploymentResult(output string) (map[string]string, error) {
	result := make(map[string]string)

	// Extract contract name
	contractNameRegex := regexp.MustCompile(`Contract name:\s+(\S+)`)
	if match := contractNameRegex.FindStringSubmatch(output); len(match) > 1 {
		result["contractName"] = match[1]
	}

	// Extract deployed address
	deployedRegex := regexp.MustCompile(`Deployed:\s+(0x[a-fA-F0-9]{40})`)
	if match := deployedRegex.FindStringSubmatch(output); len(match) > 1 {
		result["deployedAddress"] = match[1]
	}

	// Extract salt
	saltRegex := regexp.MustCompile(`Salt:\s+(0x[a-fA-F0-9]{64})`)
	if match := saltRegex.FindStringSubmatch(output); len(match) > 1 {
		result["salt"] = match[1]
	}

	// Extract initCodeHash
	initCodeHashRegex := regexp.MustCompile(`Init code hash:\s+(0x[a-fA-F0-9]{64})`)
	if match := initCodeHashRegex.FindStringSubmatch(output); len(match) > 1 {
		result["initCodeHash"] = match[1]
	}

	// Extract implementation address for proxy deployments
	implRegex := regexp.MustCompile(`Implementation:\s+(0x[a-fA-F0-9]{40})`)
	if match := implRegex.FindStringSubmatch(output); len(match) > 1 {
		result["implementationAddress"] = match[1]
	}

	// Extract Safe-specific data
	safeTxHashRegex := regexp.MustCompile(`Safe transaction hash:\s+(0x[a-fA-F0-9]{64})`)
	if match := safeTxHashRegex.FindStringSubmatch(output); len(match) > 1 {
		result["safeTxHash"] = match[1]
	}

	executedRegex := regexp.MustCompile(`Deployment status:\s+(\w+)`)
	if match := executedRegex.FindStringSubmatch(output); len(match) > 1 {
		result["status"] = match[1]
	}

	return result, nil
}

// ParsePredictionOutput parses prediction output from script
func (p *Parser) ParsePredictionOutput(output string) (*types.PredictResult, error) {
	result := &types.PredictResult{}

	// Extract predicted address
	predictedRegex := regexp.MustCompile(`Predicted:\s+(0x[a-fA-F0-9]{40})`)
	if match := predictedRegex.FindStringSubmatch(output); len(match) > 1 {
		result.Address = common.HexToAddress(match[1])
	}

	// Extract salt
	saltRegex := regexp.MustCompile(`Salt:\s+(0x[a-fA-F0-9]{64})`)
	if match := saltRegex.FindStringSubmatch(output); len(match) > 1 {
		saltBytes, _ := hex.DecodeString(strings.TrimPrefix(match[1], "0x"))
		copy(result.Salt[:], saltBytes)
	}

	return result, nil
}

// ParseLibraryAddress parses library address from deployment output
func (p *Parser) ParseLibraryAddress(output string) (common.Address, error) {
	// Look for library address pattern
	libraryRegex := regexp.MustCompile(`Deployed library:\s+(0x[a-fA-F0-9]{40})`)
	if match := libraryRegex.FindStringSubmatch(output); len(match) > 1 {
		return common.HexToAddress(match[1]), nil
	}

	// Fallback to general deployed pattern
	deployedRegex := regexp.MustCompile(`Deployed:\s+(0x[a-fA-F0-9]{40})`)
	if match := deployedRegex.FindStringSubmatch(output); len(match) > 1 {
		return common.HexToAddress(match[1]), nil
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

	if strings.Contains(outputStr, "DeploymentPendingSafe") {
		return fmt.Errorf("deployment pending Safe execution. Please execute the Safe transaction first")
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