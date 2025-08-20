package usecase

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// TagDeployment handles tagging of deployments
type TagDeployment struct {
	deploymentRepo DeploymentRepository
	progress       ProgressSink
}

// NewTagDeployment creates a new tag deployment use case
func NewTagDeployment(deploymentRepo DeploymentRepository, progress ProgressSink) *TagDeployment {
	return &TagDeployment{
		deploymentRepo: deploymentRepo,
		progress:       progress,
	}
}

// TagDeploymentParams contains parameters for tagging deployments
type TagDeploymentParams struct {
	// Identifier can be deployment ID, contract name, or address
	Identifier string
	// Tag to add or remove
	Tag string
	// Operation: "add", "remove", or "show"
	Operation string
	// ChainID for address-based lookups (optional)
	ChainID uint64
	// Namespace for filtering (optional)
	Namespace string
}

// TagDeploymentResult contains the result of a tag operation
type TagDeploymentResult struct {
	// The deployment that was tagged
	Deployment *models.Deployment
	// Operation performed
	Operation string
	// Tag that was added/removed
	Tag string
	// Current tags after operation
	CurrentTags []string
}

// Execute handles tag operations on deployments
func (t *TagDeployment) Execute(ctx context.Context, params TagDeploymentParams) (*TagDeploymentResult, error) {
	// Find the deployment
	deployment, err := t.findDeployment(ctx, params)
	if err != nil {
		return nil, err
	}

	// Handle the operation
	switch params.Operation {
	case "show":
		return t.showTags(deployment)
	case "add":
		return t.addTag(ctx, deployment, params.Tag)
	case "remove":
		return t.removeTag(ctx, deployment, params.Tag)
	default:
		return nil, fmt.Errorf("invalid operation: %s", params.Operation)
	}
}

// findDeployment locates a deployment by identifier
func (t *TagDeployment) findDeployment(ctx context.Context, params TagDeploymentParams) (*models.Deployment, error) {
	// Try as deployment ID first
	deployment, err := t.deploymentRepo.GetDeployment(ctx, params.Identifier)
	if err == nil {
		return deployment, nil
	}
	if err != domain.ErrNotFound {
		return nil, fmt.Errorf("failed to get deployment by ID: %w", err)
	}

	// If it looks like an address, try by address
	if len(params.Identifier) == 42 && params.Identifier[:2] == "0x" {
		if params.ChainID == 0 {
			return nil, fmt.Errorf("chain ID required for address-based lookup")
		}
		deployment, err = t.deploymentRepo.GetDeploymentByAddress(ctx, params.ChainID, params.Identifier)
		if err == nil {
			return deployment, nil
		}
		if err != domain.ErrNotFound {
			return nil, fmt.Errorf("failed to get deployment by address: %w", err)
		}
	}

	// Try to find by contract name
	filter := domain.DeploymentFilter{
		ContractName: params.Identifier,
		Namespace:    params.Namespace,
		ChainID:      params.ChainID,
	}
	deployments, err := t.deploymentRepo.ListDeployments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deployments) == 0 {
		return nil, fmt.Errorf("no deployment found matching '%s'", params.Identifier)
	}

	if len(deployments) == 1 {
		return deployments[0], nil
	}

	// Multiple matches - return error (interactive selection should be handled at CLI layer)
	return nil, fmt.Errorf("multiple deployments found matching '%s', please be more specific", params.Identifier)
}

// showTags displays current tags for a deployment
func (t *TagDeployment) showTags(deployment *models.Deployment) (*TagDeploymentResult, error) {
	return &TagDeploymentResult{
		Deployment:  deployment,
		Operation:   "show",
		CurrentTags: deployment.Tags,
	}, nil
}

// addTag adds a tag to a deployment
func (t *TagDeployment) addTag(ctx context.Context, deployment *models.Deployment, tag string) (*TagDeploymentResult, error) {
	// Check if tag already exists
	for _, existingTag := range deployment.Tags {
		if existingTag == tag {
			// Tag already exists, return error
			return nil, fmt.Errorf("tag '%s' already exists", tag)
		}
	}

	// Add the tag
	deployment.Tags = append(deployment.Tags, tag)

	// Save the deployment
	if err := t.deploymentRepo.SaveDeployment(ctx, deployment); err != nil {
		return nil, fmt.Errorf("failed to save deployment: %w", err)
	}

	t.progress.Info(fmt.Sprintf("Added tag '%s' to deployment %s", tag, deployment.ID))

	return &TagDeploymentResult{
		Deployment:  deployment,
		Operation:   "add",
		Tag:         tag,
		CurrentTags: deployment.Tags,
	}, nil
}

// removeTag removes a tag from a deployment
func (t *TagDeployment) removeTag(ctx context.Context, deployment *models.Deployment, tag string) (*TagDeploymentResult, error) {
	// Check if tag exists
	found := false
	newTags := make([]string, 0, len(deployment.Tags))
	for _, existingTag := range deployment.Tags {
		if existingTag == tag {
			found = true
		} else {
			newTags = append(newTags, existingTag)
		}
	}

	if !found {
		// Tag doesn't exist, return error
		return nil, fmt.Errorf("tag '%s' does not exist", tag)
	}

	// Update tags
	deployment.Tags = newTags

	// Save the deployment
	if err := t.deploymentRepo.SaveDeployment(ctx, deployment); err != nil {
		return nil, fmt.Errorf("failed to save deployment: %w", err)
	}

	t.progress.Info(fmt.Sprintf("Removed tag '%s' from deployment %s", tag, deployment.ID))

	return &TagDeploymentResult{
		Deployment:  deployment,
		Operation:   "remove",
		Tag:         tag,
		CurrentTags: deployment.Tags,
	}, nil
}

// FindDeploymentInteractive finds a deployment with support for interactive selection
func (t *TagDeployment) FindDeploymentInteractive(ctx context.Context, identifier string, chainID uint64, namespace string, selector DeploymentSelector) (*models.Deployment, error) {
	// Try exact match first
	params := TagDeploymentParams{
		Identifier: identifier,
		ChainID:    chainID,
		Namespace:  namespace,
	}
	deployment, err := t.findDeployment(ctx, params)
	if err == nil {
		return deployment, nil
	}

	// If multiple matches, use selector
	if err.Error() == fmt.Sprintf("multiple deployments found matching '%s', please be more specific", identifier) {
		// Get all matching deployments for selection
		filter := domain.DeploymentFilter{
			ContractName: identifier,
			Namespace:    namespace,
			ChainID:      chainID,
		}
		deployments, err := t.deploymentRepo.ListDeployments(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to list deployments: %w", err)
		}

		// Use selector to pick one
		return selector.SelectDeployment(ctx, deployments, "Multiple deployments found. Select one:")
	}

	return nil, err
}
