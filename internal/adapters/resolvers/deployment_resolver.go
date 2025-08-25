package resolvers

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// DeploymentResolver handles deployment resolution and selection
type DeploymentResolver struct {
	config   *config.RuntimeConfig
	repo     usecase.DeploymentRepository
	selector usecase.DeploymentSelector
}

// NewDeploymentResolver creates a new deployment resolver
func NewDeploymentResolver(
	cfg *config.RuntimeConfig,
	repo usecase.DeploymentRepository,
	selector usecase.DeploymentSelector,
) *DeploymentResolver {
	return &DeploymentResolver{
		config:   cfg,
		repo:     repo,
		selector: selector,
	}
}

// ResolveDeployment resolves a deployment reference with filtering
func (r *DeploymentResolver) ResolveDeployment(ctx context.Context, query domain.DeploymentQuery) (*models.Deployment, error) {
	// Try to find deployments matching the query
	deployments, err := r.findDeployments(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(deployments) == 0 {
		return nil, domain.ErrNotFound
	}

	if len(deployments) == 1 {
		return deployments[0], nil
	}

	// Multiple matches - use interactive selector if available
	if r.selector != nil && !r.config.NonInteractive {
		selected, err := r.selector.SelectDeployment(ctx, deployments, fmt.Sprintf("Multiple deployments found for '%s'. Select one:", query.Reference))
		if err != nil {
			return nil, fmt.Errorf("deployment selection failed: %w", err)
		}
		return selected, nil
	}

	// Non-interactive mode with multiple matches
	var suggestions []string
	for _, dep := range deployments {
		suggestion := fmt.Sprintf("  - %s (chain:%d/%s/%s at %s)",
			dep.ID, dep.ChainID, dep.Namespace, dep.ContractDisplayName(), dep.Address)
		suggestions = append(suggestions, suggestion)
	}
	sort.Strings(suggestions)
	return nil, fmt.Errorf("multiple deployments found matching '%s', please be more specific:\n%s",
		query.Reference, strings.Join(suggestions, "\n"))
}

// findDeployments finds deployments matching the query
func (r *DeploymentResolver) findDeployments(ctx context.Context, query domain.DeploymentQuery) ([]*models.Deployment, error) {
	ref := query.Reference

	// Get default values from runtime config if not specified in query
	namespace := query.Namespace
	if namespace == "" {
		namespace = r.config.Namespace
	}

	chainID := query.ChainID
	if chainID == 0 && r.config.Network != nil {
		chainID = r.config.Network.ChainID
	}

	// 1. Try as deployment ID
	deployment, err := r.repo.GetDeployment(ctx, ref)
	if err == nil {
		return []*models.Deployment{deployment}, nil
	}

	// 2. Try as address (starts with 0x and is 42 chars)
	if strings.HasPrefix(ref, "0x") && len(ref) == 42 {
		if chainID != 0 {
			// If chain ID is specified, try direct lookup
			deployment, err = r.repo.GetDeploymentByAddress(ctx, chainID, ref)
			if err == nil {
				return []*models.Deployment{deployment}, nil
			}
		} else {
			// Search all deployments for this address
			deployments, err := r.repo.ListDeployments(ctx, domain.DeploymentFilter{})
			if err != nil {
				return nil, fmt.Errorf("failed to list deployments: %w", err)
			}

			var matches []*models.Deployment
			for _, dep := range deployments {
				if strings.EqualFold(dep.Address, ref) {
					matches = append(matches, dep)
				}
			}
			if len(matches) > 0 {
				return matches, nil
			}
		}
	}

	// 3. Parse the reference to extract components
	contractName, label, extractedNamespace, extractedChainID := r.parseReference(ref)

	// Override with extracted values if found
	if extractedNamespace != "" {
		namespace = extractedNamespace
	}
	if extractedChainID != 0 {
		chainID = extractedChainID
	}

	// 4. Build filter and search
	filter := domain.DeploymentFilter{
		ContractName: contractName,
		Label:        label,
		ChainID:      chainID,
		Namespace:    namespace,
	}

	deployments, err := r.repo.ListDeployments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	return deployments, nil
}

// parseReference parses a deployment reference and extracts components
// Supports formats:
// - Contract name: "Counter"
// - Contract with label: "Counter:v2"
// - Namespace/contract: "staging/Counter"
// - Chain/contract: "11155111/Counter"
// - Namespace/chain/contract: "staging/11155111/Counter"
// - Namespace/chain/contract:label: "staging/11155111/Counter:v2"
func (r *DeploymentResolver) parseReference(ref string) (contractName, label, namespace string, chainID uint64) {
	// First extract label if present
	parts := strings.Split(ref, ":")
	base := parts[0]
	if len(parts) > 1 {
		label = parts[1]
	}

	// Now parse the base part
	segments := strings.Split(base, "/")

	switch len(segments) {
	case 1:
		// Just contract name
		contractName = segments[0]
	case 2:
		// Could be namespace/contract or chainID/contract
		if cid := parseChainID(segments[0]); cid != 0 {
			chainID = cid
			contractName = segments[1]
		} else {
			namespace = segments[0]
			contractName = segments[1]
		}
	case 3:
		// namespace/chainID/contract
		namespace = segments[0]
		if cid := parseChainID(segments[1]); cid != 0 {
			chainID = cid
		}
		contractName = segments[2]
	}

	return contractName, label, namespace, chainID
}

// parseChainID tries to parse a string as a chain ID
func parseChainID(s string) uint64 {
	chainID, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return chainID
}

// Ensure the adapter implements the interface
var _ usecase.DeploymentResolver = (*DeploymentResolver)(nil)

