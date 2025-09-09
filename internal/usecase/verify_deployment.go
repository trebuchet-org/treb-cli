package usecase

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// VerifyDeployment handles contract verification on block explorers
type VerifyDeployment struct {
	repo             DeploymentRepository
	contractVerifier ContractVerifier
	networkResolver  NetworkResolver
}

// NewVerifyDeployment creates a new verify deployment use case
func NewVerifyDeployment(
	repo DeploymentRepository,
	contractVerifier ContractVerifier,
	networkResolver NetworkResolver,
) *VerifyDeployment {
	return &VerifyDeployment{
		repo:             repo,
		contractVerifier: contractVerifier,
		networkResolver:  networkResolver,
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
	Deployment *models.Deployment
	Success    bool
	Errors     []string
}

// VerifyAll verifies all unverified deployments
func (v *VerifyDeployment) VerifyAll(ctx context.Context, options VerifyOptions) (*VerifyAllResult, error) {
	// Get all deployments
	deployments, err := v.repo.ListDeployments(ctx, domain.DeploymentFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	result := &VerifyAllResult{
		ToVerify: make([]*models.Deployment, 0),
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
		tx, err := v.repo.GetTransaction(ctx, deployment.TransactionID)
		if err != nil {
			result.Skipped = append(result.Skipped, &SkippedDeployment{
				Deployment: deployment,
				Reason:     "Transaction not found",
			})
			continue
		}

		if tx.Status != models.TransactionStatusExecuted {
			result.Skipped = append(result.Skipped, &SkippedDeployment{
				Deployment: deployment,
				Reason:     fmt.Sprintf("Transaction %s", tx.Status),
			})
			continue
		}

		// Check if should verify
		if options.Force || shouldVerify(deployment) {
			result.ToVerify = append(result.ToVerify, deployment)
		} else if deployment.Verification.Status == models.VerificationStatusVerified {
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
	var deployment *models.Deployment
	var err error

	// Check if identifier is an address
	if strings.HasPrefix(identifier, "0x") && len(identifier) == 42 {
		if filter.ChainID == 0 {
			return nil, fmt.Errorf("--network flag is required when looking up by address")
		}
		deployment, err = v.repo.GetDeploymentByAddress(ctx, filter.ChainID, identifier)
		if err != nil {
			return nil, fmt.Errorf("deployment not found at address %s on network", identifier)
		}
	} else {
		// Find deployment by identifier
		deployment, err = v.findDeploymentByIdentifier(ctx, identifier, filter)
		if err != nil {
			return nil, err
		}
	}

	// Check if already verified
	if deployment.Verification.Status == models.VerificationStatusVerified && !options.Force {
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
func (v *VerifyDeployment) verifyDeployment(ctx context.Context, deployment *models.Deployment, options VerifyOptions) *VerifyResult {
	// Get network name for the deployment's chain ID
	networkName, err := v.getNetworkNameForChainID(ctx, deployment.ChainID)
	if err != nil {
		return &VerifyResult{
			Deployment: deployment,
			Success:    false,
			Errors:     []string{fmt.Sprintf("failed to resolve network for chain ID %d: %v", deployment.ChainID, err)},
		}
	}
	
	// Get network info
	networkInfo, err := v.networkResolver.ResolveNetwork(ctx, networkName)
	if err != nil {
		return &VerifyResult{
			Deployment: deployment,
			Success:    false,
			Errors:     []string{fmt.Sprintf("failed to resolve network %s: %v", networkName, err)},
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
	if err := v.repo.SaveDeployment(ctx, deployment); err != nil {
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
func (v *VerifyDeployment) findDeploymentByIdentifier(ctx context.Context, identifier string, filter domain.DeploymentFilter) (*models.Deployment, error) {
	// Get all deployments with filter
	deployments, err := v.repo.ListDeployments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	// Look for matches
	matches := make([]*models.Deployment, 0)
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
func matchesIdentifier(d *models.Deployment, identifier string, parts []string) bool {
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
	}

	// Match full deployment ID
	return d.ID == identifier
}

// shouldVerify checks if a deployment should be verified
func shouldVerify(deployment *models.Deployment) bool {
	status := deployment.Verification.Status
	return status == models.VerificationStatusFailed ||
		status == models.VerificationStatusPartial ||
		status == models.VerificationStatusUnverified ||
		status == ""
}

// parseChainID tries to parse a string as a chain ID
func parseChainID(s string) uint64 {
	chainID, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return chainID
}

// getNetworkNameForChainID attempts to find a network name for a chain ID
func (v *VerifyDeployment) getNetworkNameForChainID(ctx context.Context, chainID uint64) (string, error) {
	// Get all available network names
	networkNames := v.networkResolver.GetNetworks(ctx)
	
	// Try to resolve each network to find matching chain ID
	for _, name := range networkNames {
		network, err := v.networkResolver.ResolveNetwork(ctx, name)
		if err != nil {
			// Skip networks that can't be resolved
			continue
		}
		
		if network.ChainID == chainID {
			return name, nil
		}
	}
	
	// If no network found, return an error
	return "", fmt.Errorf("no network configuration found for chain ID %d. Please ensure the network is configured in foundry.toml", chainID)
}

// Result types

// VerifyAllResult contains the result of verifying all deployments
type VerifyAllResult struct {
	ToVerify     []*models.Deployment
	Skipped      []*SkippedDeployment
	Results      []*VerifyResult
	SuccessCount int
}

// SkippedDeployment represents a deployment that was skipped
type SkippedDeployment struct {
	Deployment *models.Deployment
	Reason     string
}
