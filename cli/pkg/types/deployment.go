package types

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
)

type Status string

const (
	StatusExecuted Status = "EXECUTED"
	StatusQueued   Status = "PENDING_SAFE"
	StatusUnknown  Status = "UNKNOWN"
)

type DeploymentType string

const (
	SingletonDeployment DeploymentType = "SINGLETON"
	ProxyDeployment     DeploymentType = "PROXY"
	LibraryDeployment   DeploymentType = "LIBRARY"
	UnknownDeployment   DeploymentType = "UNKNOWN"
)

type DeployStrategy string

const (
	Create2Strategy DeployStrategy = "CREATE2"
	Create3Strategy DeployStrategy = "CREATE3"
	UnknownStrategy DeployStrategy = "UNKNOWN"
)

func ParseStatus(status string) (Status, error) {
	switch status {
	case "EXECUTED":
		return StatusExecuted, nil
	case "PENDING_SAFE":
		return StatusQueued, nil
	default:
		return StatusUnknown, fmt.Errorf("unknown status: %s", status)
	}
}

func ParseDeploymentType(deploymentType string) (DeploymentType, error) {
	switch deploymentType {
	case "SINGLETON":
		return SingletonDeployment, nil
	case "PROXY":
		return ProxyDeployment, nil
	case "LIBRARY":
		return LibraryDeployment, nil
	default:
		return UnknownDeployment, fmt.Errorf("unknown deployment type: %s", deploymentType)
	}
}

type DeploymentEntry struct {
	FQID            string         `json:"fqid"` // Fully qualified identifier
	ShortID         string         `json:"sid"`  // Short identifier
	Address         common.Address `json:"address"`
	ContractName    string         `json:"contract_name"`
	Namespace       string         `json:"namespace"`
	Type            DeploymentType `json:"type"`                       // implementation/proxy
	Salt            string         `json:"salt"`                       // hex string
	InitCodeHash    string         `json:"init_code_hash"`             // hex string
	ConstructorArgs string         `json:"constructor_args,omitempty"` // Raw hex-encoded constructor args

	// Label for all deployments (optional for implementations, required for proxies)
	Label string `json:"label,omitempty"`

	// Proxy-specific fields
	TargetDeploymentFQID string `json:"target_deployment_fqid,omitempty"` // For proxy deployments

	// Version tags (metadata only, not part of salt)
	Tags []string `json:"tags,omitempty"`

	Verification Verification         `json:"verification"`
	Deployment   DeploymentInfo       `json:"deployment"`
	Metadata     ContractMetadata     `json:"metadata"`
	Target       *DeploymentEntry     `json:"-"`
	NetworkInfo  *network.NetworkInfo `json:"-"`
}

type Verification struct {
	Status      string                    `json:"status"` // verified/pending/failed/partial
	ExplorerUrl string                    `json:"explorer_url,omitempty"`
	Reason      string                    `json:"reason,omitempty"`
	Verifiers   map[string]VerifierStatus `json:"verifiers,omitempty"` // etherscan, sourcify status
}

type VerifierStatus struct {
	Status string `json:"status"` // verified/pending/failed
	URL    string `json:"url,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type DeploymentInfo struct {
	TxHash        *common.Hash `json:"tx_hash,omitempty"`
	SafeTxHash    *common.Hash `json:"safe_tx_hash,omitempty"`
	BlockNumber   uint64       `json:"block_number,omitempty"`
	BroadcastFile string       `json:"broadcast_file"`
	Timestamp     time.Time    `json:"timestamp"`
	Status        Status       `json:"status"` // "deployed", "pending_safe"
	SafeAddress   string       `json:"safe_address,omitempty"`
	SafeNonce     uint64       `json:"safe_nonce,omitempty"`
	Deployer      string       `json:"deployer,omitempty"` // Address that deployed the contract
}

type ContractMetadata struct {
	SourceCommit string                 `json:"source_commit"`
	Compiler     string                 `json:"compiler"`
	SourceHash   string                 `json:"source_hash"`
	ContractPath string                 `json:"contract_path"`   // Full path like ./src/Contract.sol:Contract
	ScriptPath   string                 `json:"script_path"`     // Deployment script path like script/deploy/DeployCounter.s.sol
	Extra        map[string]interface{} `json:"extra,omitempty"` // Additional metadata (e.g., proxy type, implementation address)
}

// GetDisplayName returns a human-friendly name for the deployment
func (d *DeploymentEntry) GetDisplayName() string {
	switch d.Type {
	case ProxyDeployment:
		if d.Target != nil {
			displayName := fmt.Sprintf("%s:%s", d.ContractName, d.Target.GetDisplayName())
			if d.Label != "" {
				return fmt.Sprintf("%s:%s", displayName, d.Label)
			}
			return displayName
		} else {
			return d.ShortID
		}
	default:
		return d.ShortID
	}
}

// GetColoredDisplayName returns a colored version of the display name for use in tables
// - ContractName is default (white/bold)
// - "Proxy" suffix and ":" separators are faint
// - Labels are cyan
func (d *DeploymentEntry) GetColoredDisplayName() string {
	// Create color styles
	faintStyle := color.New(color.Faint)
	cyanStyle := color.New(color.FgCyan)

	switch d.Type {
	case ProxyDeployment:
		// Build colored proxy name
		result := faintStyle.Sprint(d.ContractName)

		// Recursively add target display name
		if d.Target != nil {
			result += faintStyle.Sprint(":")
			// Get the target's colored display name recursively
			targetDisplay := d.Target.ContractName
			result += targetDisplay
		}

		// Add label if present
		if d.Label != "" {
			result += faintStyle.Sprint(":")
			result += cyanStyle.Sprint(d.Label)
		}

		return result
	default:
		// For non-proxy deployments
		result := d.ContractName

		// Add label if present
		if d.Label != "" {
			result += faintStyle.Sprint(":")
			result += cyanStyle.Sprint(d.Label)
		}

		return result
	}
}
