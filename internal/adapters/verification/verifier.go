package verification

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
	"github.com/trebuchet-org/treb-cli/cli/pkg/verification"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// VerifierAdapter wraps the existing verification.Manager to implement ContractVerifier
type VerifierAdapter struct {
	verificationManager *verification.Manager
}

// NewVerifierAdapter creates a new adapter wrapping the existing verification manager
func NewVerifierAdapter(cfg *config.RuntimeConfig) (*VerifierAdapter, error) {
	// Create registry manager
	registryManager, err := registry.NewManager(cfg.ProjectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry manager: %w", err)
	}

	// Create network resolver
	networkResolver, err := network.NewResolver(cfg.ProjectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create network resolver: %w", err)
	}

	// Create verification manager
	verificationManager := verification.NewManager(registryManager, networkResolver)

	return &VerifierAdapter{
		verificationManager: verificationManager,
	}, nil
}

// Verify performs contract verification on multiple verifiers
func (v *VerifierAdapter) Verify(ctx context.Context, deployment *domain.Deployment, network *domain.NetworkInfo) error {
	// Convert domain deployment to legacy types
	legacyDeployment := convertToLegacyDeployment(deployment)
	
	// Perform verification (the legacy manager will update the deployment directly)
	err := v.verificationManager.VerifyDeployment(legacyDeployment)
	
	// Copy verification results back to domain deployment
	deployment.Verification = convertToDomainVerificationInfo(&legacyDeployment.Verification)
	
	// Return error if verification failed
	if err != nil {
		// Check if it's a partial success (some verifiers succeeded)
		if deployment.Verification.Status == domain.VerificationStatusPartial {
			// Don't return error for partial success
			return nil
		}
		return err
	}
	
	return nil
}

// GetVerificationStatus retrieves the current verification status
func (v *VerifierAdapter) GetVerificationStatus(ctx context.Context, deployment *domain.Deployment) (*domain.VerificationInfo, error) {
	// This would typically check the actual on-chain verification status
	// For now, we just return the stored status
	return &deployment.Verification, nil
}

// convertToLegacyDeployment converts domain deployment to legacy types
func convertToLegacyDeployment(dep *domain.Deployment) *types.Deployment {
	if dep == nil {
		return nil
	}

	legacyDep := &types.Deployment{
		ID:            dep.ID,
		Namespace:     dep.Namespace,
		ChainID:       dep.ChainID,
		ContractName:  dep.ContractName,
		Label:         dep.Label,
		Address:       dep.Address,
		Type:          types.DeploymentType(dep.Type),
		TransactionID: dep.TransactionID,
		DeploymentStrategy: types.DeploymentStrategy{
			Method:          types.DeploymentMethod(dep.DeploymentStrategy.Method),
			Salt:            dep.DeploymentStrategy.Salt,
			InitCodeHash:    dep.DeploymentStrategy.InitCodeHash,
			Factory:         dep.DeploymentStrategy.Factory,
			ConstructorArgs: dep.DeploymentStrategy.ConstructorArgs,
			Entropy:         dep.DeploymentStrategy.Entropy,
		},
		Artifact: types.ArtifactInfo{
			Path:            dep.Artifact.Path,
			CompilerVersion: dep.Artifact.CompilerVersion,
			BytecodeHash:    dep.Artifact.BytecodeHash,
			ScriptPath:      dep.Artifact.ScriptPath,
			GitCommit:       dep.Artifact.GitCommit,
		},
		Verification: types.VerificationInfo{
			Status:       types.VerificationStatus(dep.Verification.Status),
			EtherscanURL: dep.Verification.EtherscanURL,
			VerifiedAt:   dep.Verification.VerifiedAt,
			Reason:       dep.Verification.Reason,
		},
		Tags:      dep.Tags,
		CreatedAt: dep.CreatedAt,
		UpdatedAt: dep.UpdatedAt,
	}

	// Convert proxy info if present
	if dep.ProxyInfo != nil {
		legacyDep.ProxyInfo = &types.ProxyInfo{
			Type:           dep.ProxyInfo.Type,
			Implementation: dep.ProxyInfo.Implementation,
			Admin:          dep.ProxyInfo.Admin,
		}
		// Convert history
		for _, upgrade := range dep.ProxyInfo.History {
			legacyDep.ProxyInfo.History = append(legacyDep.ProxyInfo.History, types.ProxyUpgrade{
				ImplementationID: upgrade.ImplementationID,
				UpgradedAt:       upgrade.UpgradedAt,
				UpgradeTxID:      upgrade.UpgradeTxID,
			})
		}
	}

	// Convert verifiers if present
	if dep.Verification.Verifiers != nil {
		legacyDep.Verification.Verifiers = make(map[string]types.VerifierStatus)
		for name, status := range dep.Verification.Verifiers {
			legacyDep.Verification.Verifiers[name] = types.VerifierStatus{
				Status: status.Status,
				URL:    status.URL,
				Reason: status.Reason,
			}
		}
	}

	return legacyDep
}

// convertToDomainVerificationInfo converts legacy verification info to domain
func convertToDomainVerificationInfo(info *types.VerificationInfo) domain.VerificationInfo {
	domainInfo := domain.VerificationInfo{
		Status:       domain.VerificationStatus(info.Status),
		EtherscanURL: info.EtherscanURL,
		VerifiedAt:   info.VerifiedAt,
		Reason:       info.Reason,
	}

	// Convert verifiers if present
	if info.Verifiers != nil {
		domainInfo.Verifiers = make(map[string]domain.VerifierStatus)
		for name, status := range info.Verifiers {
			domainInfo.Verifiers[name] = domain.VerifierStatus{
				Status: status.Status,
				URL:    status.URL,
				Reason: status.Reason,
			}
		}
	}

	return domainInfo
}

// Ensure the adapter implements the interface
var _ usecase.ContractVerifier = (*VerifierAdapter)(nil)