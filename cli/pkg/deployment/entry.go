package deployment

import (
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// BuildDeploymentEntry builds a complete DeploymentEntry from ParsedDeploymentResult
func (d *DeploymentContext) BuildDeploymentEntry(result *ParsedDeploymentResult) (*types.DeploymentEntry, error) {
	// Calculate init code hash
	initCodeHash := ""
	if len(result.InitCode) > 0 {
		hash := sha256.Sum256(result.InitCode)
		initCodeHash = "0x" + hex.EncodeToString(hash[:])
	}

	// Create the deployment entry
	entry := &types.DeploymentEntry{
		FQID:         d.GetFQID(),
		ShortID:      d.GetShortID(),
		Address:      result.Deployed,
		ContractName: d.contractInfo.Name,
		Namespace:    d.Params.Namespace,
		Type:         result.ParsedDeploymentType,
		Salt:         common.Hash(result.Salt).Hex(),
		InitCodeHash: initCodeHash,
		Label:        d.Params.Label,
		NetworkInfo:  d.networkInfo,
	}

	// Add constructor args if present
	if len(result.ConstructorArgs) > 0 {
		entry.ConstructorArgs = "0x" + hex.EncodeToString(result.ConstructorArgs)
	}

	// For proxy deployments, set the target deployment FQID
	if result.ParsedDeploymentType == types.ProxyDeployment && d.targetDeploymentFQID != "" {
		entry.TargetDeploymentFQID = d.targetDeploymentFQID
	}

	// Set verification status
	entry.Verification = types.Verification{
		Status: "pending",
	}

	// Set deployment info
	entry.Deployment = types.DeploymentInfo{
		BlockNumber:   result.BlockNumber,
		BroadcastFile: result.BroadcastFile,
		Timestamp:     time.Now(),
		Status:        result.ParsedStatus,
	}

	// Add transaction hash if executed
	if result.ParsedStatus == types.StatusExecuted && result.TxHash != (common.Hash{}) {
		entry.Deployment.TxHash = &result.TxHash
	}

	// Add Safe transaction hash if pending
	if result.ParsedStatus == types.StatusQueued && result.SafeTxHash != ([32]byte{}) {
		safeTxHash := common.Hash(result.SafeTxHash)
		entry.Deployment.SafeTxHash = &safeTxHash
		// TODO: Add Safe address and nonce from deployment config
		if d.executorConfig != nil && d.executorConfig.SenderType == abi.SenderTypeSafe {
			entry.Deployment.SafeAddress = d.executorConfig.Sender.Hex()
		}
	}

	// Set deployer address
	if d.executorConfig != nil {
		entry.Deployment.Deployer = d.executorConfig.Sender.Hex()
	}

	// Set contract metadata
	entry.Metadata = types.ContractMetadata{
		SourceCommit: getGitCommit(),
		Compiler:     getCompilerVersion(d.contractInfo),
		SourceHash:   d.contractInfo.GetSourceHash(),
		ContractPath: d.contractInfo.Path,
		ScriptPath:   d.ScriptPath,
	}

	// Add extra metadata based on deployment type
	if result.ParsedDeploymentType == types.ProxyDeployment {
		if entry.Metadata.Extra == nil {
			entry.Metadata.Extra = make(map[string]interface{})
		}
		// Add proxy-specific metadata
		if d.implementationInfo != nil {
			entry.Metadata.Extra["implementation_contract"] = d.implementationInfo.Name
			entry.Metadata.Extra["implementation_path"] = d.implementationInfo.Path
		}
	}

	return entry, nil
}

// getGitCommit returns the current git commit hash if available
func getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getCompilerVersion extracts the compiler version from contract info
func getCompilerVersion(contractInfo *contracts.ContractInfo) string {
	if contractInfo == nil || contractInfo.Artifact == nil {
		return ""
	}
	return contractInfo.Artifact.Metadata.Compiler.Version
}

// Data sources that are missing and would need to be added:
// 1. Git commit hash - needs git integration
// 2. Safe nonce - needs to query Safe contract or track locally
// 3. Actual gas used and gas price - currently available in DeploymentResult
// 4. More detailed verification status - needs integration with verification services
// 5. Libraries used by the contract - available in contractInfo.GetRequiredLibraries()