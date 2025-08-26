package verification

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// VerifierAdapter wraps the internal verifier to implement ContractVerifier
type VerifierAdapter struct {
	verifier *InternalVerifier
}

// NewVerifierAdapter creates a new adapter wrapping the internal verifier
func NewVerifierAdapter(cfg *config.RuntimeConfig) (*VerifierAdapter, error) {
	// Create internal verifier
	verifier, err := NewInternalVerifier(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create internal verifier: %w", err)
	}

	return &VerifierAdapter{
		verifier: verifier,
	}, nil
}

// Verify performs contract verification on multiple verifiers
func (v *VerifierAdapter) Verify(ctx context.Context, deployment *models.Deployment, network *config.Network) error {
	return v.verifier.Verify(ctx, deployment, network)
}

// GetVerificationStatus retrieves the current verification status
func (v *VerifierAdapter) GetVerificationStatus(ctx context.Context, deployment *models.Deployment) (*models.VerificationInfo, error) {
	return v.verifier.GetVerificationStatus(ctx, deployment)
}

// Ensure the adapter implements the interface
var _ usecase.ContractVerifier = (*VerifierAdapter)(nil)
