package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// VerifyDeployment handles contract verification on block explorers
type VerifyDeployment struct {
	deploymentStore  DeploymentStore
	contractVerifier ContractVerifier
	networkResolver  NetworkResolver
	transactionStore TransactionStore
}

// NewVerifyDeployment creates a new verify deployment use case
func NewVerifyDeployment(
	deploymentStore DeploymentStore,
	contractVerifier ContractVerifier,
	networkResolver NetworkResolver,
	transactionStore TransactionStore,
) *VerifyDeployment {
	return &VerifyDeployment{
		deploymentStore:  deploymentStore,
		contractVerifier: contractVerifier,
		networkResolver:  networkResolver,
		transactionStore: transactionStore,
	}
}

// VerifyOptions contains options for verification
type VerifyOptions struct {
	Force        bool   // Re-verify even if already verified
	ContractPath string // Override contract path
	Debug        bool   // Show debug information
}

// VerifyResult contains the result of verification
type VerifyResult struct {
	Deployment *domain.Deployment
	Success    bool
	Errors     []string
}

// VerifyAll verifies all unverified deployments
func (v *VerifyDeployment) VerifyAll(ctx context.Context, options VerifyOptions) (*VerifyAllResult, error) {
	// Get all deployments
	deployments, err := v.deploymentStore.ListDeployments(ctx, domain.DeploymentFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	result := &VerifyAllResult{
		ToVerify: make([]*domain.Deployment, 0),
		Skipped:  make([]*SkippedDeployment, 0),
		Results:  make([]*VerifyResult, 0),
	}

	// Filter deployments
	for _, deployment := range deployments {
		// Skip local chain deployments
		if deployment.ChainID == 31337 {
			result.Skipped = append(result.Skipped, &SkippedDeployment{
				Deployment: deployment,
				Reason:     "Local chain",
			})
			continue
		}

		// Skip deployments without transaction
		if deployment.TransactionID == "" {
			result.Skipped = append(result.Skipped, &SkippedDeployment{
				Deployment: deployment,
				Reason:     "No transaction ID",
			})
			continue
		}

		// Check transaction status
		tx, err := v.transactionStore.GetTransaction(ctx, deployment.TransactionID)
		if err != nil {
			result.Skipped = append(result.Skipped, &SkippedDeployment{
				Deployment: deployment,
				Reason:     "Transaction not found",
			})
			continue
		}

		if tx.Status != domain.TransactionStatusExecuted {
			result.Skipped = append(result.Skipped, &SkippedDeployment{
				Deployment: deployment,
				Reason:     fmt.Sprintf("Transaction %s", tx.Status),
			})
			continue
		}

		// Check if should verify
		if options.Force || shouldVerify(deployment) {
			result.ToVerify = append(result.ToVerify, deployment)
		} else if deployment.Verification.Status == domain.VerificationStatusVerified {
			result.Skipped = append(result.Skipped, &SkippedDeployment{
				Deployment: deployment,
				Reason:     "Already verified",
			})
		}
	}

	// Verify each deployment
	for _, deployment := range result.ToVerify {
		verifyResult := v.verifyDeployment(ctx, deployment, options)
		result.Results = append(result.Results, verifyResult)
		if verifyResult.Success {
			result.SuccessCount++
		}
	}

	return result, nil
}

// VerifySpecific verifies a specific deployment
func (v *VerifyDeployment) VerifySpecific(ctx context.Context, identifier string, filter domain.DeploymentFilter, options VerifyOptions) (*VerifyResult, error) {
	var deployment *domain.Deployment
	var err error

	// Check if identifier is an address
	if strings.HasPrefix(identifier, "0x") && len(identifier) == 42 {
		if filter.ChainID == 0 {
			return nil, fmt.Errorf("--chain flag is required when looking up by address")
		}
		deployment, err = v.deploymentStore.GetDeploymentByAddress(ctx, filter.ChainID, identifier)
		if err != nil {
			return nil, fmt.Errorf("deployment not found at address %s on chain %d", identifier, filter.ChainID)
		}
	} else {
		// Find deployment by identifier
		deployment, err = v.findDeploymentByIdentifier(ctx, identifier, filter)
		if err != nil {
			return nil, err
		}
	}

	// Check if already verified
	if deployment.Verification.Status == domain.VerificationStatusVerified && !options.Force {
		return &VerifyResult{
			Deployment: deployment,
			Success:    true,
			Errors:     []string{"Already verified. Use --force to re-verify."},
		}, nil
	}

	// Handle contract path override
	originalPath := ""
	if options.ContractPath != "" {
		originalPath = deployment.Artifact.Path
		deployment.Artifact.Path = options.ContractPath
	}

	// Verify the deployment
	result := v.verifyDeployment(ctx, deployment, options)

	// Restore original path if verification failed
	if !result.Success && originalPath != "" {
		deployment.Artifact.Path = originalPath
	}

	return result, nil
}

// verifyDeployment performs the actual verification
func (v *VerifyDeployment) verifyDeployment(ctx context.Context, deployment *domain.Deployment, options VerifyOptions) *VerifyResult {
	// Get network info
	networkInfo, err := v.networkResolver.ResolveNetwork(ctx, getNetworkName(deployment.ChainID))
	if err != nil {
		return &VerifyResult{
			Deployment: deployment,
			Success:    false,
			Errors:     []string{fmt.Sprintf("failed to resolve network: %v", err)},
		}
	}

	// Perform verification
	err = v.contractVerifier.Verify(ctx, deployment, networkInfo)
	if err != nil {
		return &VerifyResult{
			Deployment: deployment,
			Success:    false,
			Errors:     []string{err.Error()},
		}
	}

	// Save updated deployment
	if err := v.deploymentStore.SaveDeployment(ctx, deployment); err != nil {
		return &VerifyResult{
			Deployment: deployment,
			Success:    false,
			Errors:     []string{fmt.Sprintf("failed to update registry: %v", err)},
		}
	}

	return &VerifyResult{
		Deployment: deployment,
		Success:    true,
	}
}

// findDeploymentByIdentifier finds a deployment by various identifier formats
func (v *VerifyDeployment) findDeploymentByIdentifier(ctx context.Context, identifier string, filter domain.DeploymentFilter) (*domain.Deployment, error) {
	// Get all deployments with filter
	deployments, err := v.deploymentStore.ListDeployments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	// Look for matches
	matches := make([]*domain.Deployment, 0)
	parts := strings.Split(identifier, "/")

	for _, d := range deployments {
		if matchesIdentifier(d, identifier, parts) {
			matches = append(matches, d)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no deployments found matching '%s'", identifier)
	} else if len(matches) == 1 {
		return matches[0], nil
	} else {
		// Multiple matches - in the future, we could use an interactive selector here
		return nil, fmt.Errorf("multiple deployments found matching '%s', please be more specific", identifier)
	}
}

// matchesIdentifier checks if a deployment matches the given identifier
func matchesIdentifier(d *domain.Deployment, identifier string, parts []string) bool {
	// Simple match: contract name or contract:label
	shortID := d.ContractName
	if d.Label != "" {
		shortID = fmt.Sprintf("%s:%s", d.ContractName, d.Label)
	}
	if d.ContractName == identifier || shortID == identifier {
		return true
	}

	// Match namespace/contract or namespace/contract:label
	if len(parts) == 2 {
		namespace := parts[0]
		contractPart := parts[1]

		// Check if first part is a namespace
		if d.Namespace == namespace && (d.ContractName == contractPart || shortID == contractPart) {
			return true
		}

		// Check if first part is a chain ID
		if chainID := parseChainID(parts[0]); chainID != 0 {
			if d.ChainID == chainID && (d.ContractName == contractPart || shortID == contractPart) {
				return true
			}
		}
	}

	// Match namespace/chain/contract
	if len(parts) == 3 {
		if d.Namespace == parts[0] {
			if chainID := parseChainID(parts[1]); chainID != 0 {
				if d.ChainID == chainID && (d.ContractName == parts[2] || shortID == parts[2]) {
					return true
				}
			}
		}
	}

	// Match full deployment ID
	return d.ID == identifier
}

// shouldVerify checks if a deployment should be verified
func shouldVerify(deployment *domain.Deployment) bool {
	status := deployment.Verification.Status
	return status == domain.VerificationStatusFailed ||
		status == domain.VerificationStatusPartial ||
		status == domain.VerificationStatusUnverified ||
		status == ""
}

// getNetworkName returns network name for a chain ID
func getNetworkName(chainID uint64) string {
	// This is a simplified version - the actual implementation
	// would use the network resolver
	switch chainID {
	case 1:
		return "mainnet"
	case 11155111:
		return "sepolia"
	case 137:
		return "polygon"
	case 10:
		return "optimism"
	case 42161:
		return "arbitrum"
	case 8453:
		return "base"
	default:
		return fmt.Sprintf("chain-%d", chainID)
	}
}

// Result types

// VerifyAllResult contains the result of verifying all deployments
type VerifyAllResult struct {
	ToVerify     []*domain.Deployment
	Skipped      []*SkippedDeployment
	Results      []*VerifyResult
	SuccessCount int
}

// SkippedDeployment represents a deployment that was skipped
type SkippedDeployment struct {
	Deployment *domain.Deployment
	Reason     string
}

