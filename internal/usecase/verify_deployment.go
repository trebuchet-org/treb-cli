package usecase

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// VerifyDeployment handles contract verification on block explorers
type VerifyDeployment struct {
	repo               DeploymentRepository
	contractVerifier   ContractVerifier
	networkResolver    NetworkResolver
	deploymentResolver DeploymentResolver
	progress           ProgressSink
}

// NewVerifyDeployment creates a new verify deployment use case
func NewVerifyDeployment(
	repo DeploymentRepository,
	contractVerifier ContractVerifier,
	networkResolver NetworkResolver,
	deploymentResolver DeploymentResolver,
	progress ProgressSink,
) *VerifyDeployment {
	return &VerifyDeployment{
		repo:               repo,
		contractVerifier:   contractVerifier,
		networkResolver:    networkResolver,
		deploymentResolver: deploymentResolver,
		progress:           progress,
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
func (v *VerifyDeployment) VerifyAll(ctx context.Context, filter domain.DeploymentFilter, options VerifyOptions) (*VerifyAllResult, error) {
	v.progress.OnProgress(ctx, ProgressEvent{
		Stage:   "gathering",
		Message: "Gathering deployments to verify...",
		Spinner: true,
	})

	// Get all deployments matching the filter
	deployments, err := v.repo.ListDeployments(ctx, filter)
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
	for i, deployment := range result.ToVerify {
		v.progress.OnProgress(ctx, ProgressEvent{
			Stage:   "verifying",
			Current: i + 1,
			Total:   len(result.ToVerify),
			Message: fmt.Sprintf("Verifying %s on chain %d...", deployment.ContractName, deployment.ChainID),
			Spinner: true,
		})

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
	// Use the deployment resolver for consistent deployment resolution with interactive selection
	query := domain.DeploymentQuery{
		Reference: identifier,
		ChainID:   filter.ChainID,
		Namespace: filter.Namespace,
	}

	deployment, err := v.deploymentResolver.ResolveDeployment(ctx, query)
	if err != nil {
		return nil, err
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
	// Report that we're starting verification for this specific contract
	displayName := deployment.ContractName
	if deployment.Label != "" {
		displayName = fmt.Sprintf("%s:%s", deployment.ContractName, deployment.Label)
	}

	v.progress.OnProgress(ctx, ProgressEvent{
		Stage:   "network-resolve",
		Message: fmt.Sprintf("Resolving network for %s...", displayName),
		Spinner: true,
	})

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

	v.progress.OnProgress(ctx, ProgressEvent{
		Stage:   "verification",
		Message: fmt.Sprintf("Submitting %s to block explorers...", displayName),
		Spinner: true,
	})

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

// shouldVerify checks if a deployment should be verified
func shouldVerify(deployment *models.Deployment) bool {
	status := deployment.Verification.Status
	return status == models.VerificationStatusFailed ||
		status == models.VerificationStatusPartial ||
		status == models.VerificationStatusUnverified ||
		status == ""
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
