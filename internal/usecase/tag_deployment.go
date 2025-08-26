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
	resolver       DeploymentResolver
	progress       ProgressSink
}

// NewTagDeployment creates a new tag deployment use case
func NewTagDeployment(
	deploymentRepo DeploymentRepository,
	resolver DeploymentResolver,
	progress ProgressSink,
) *TagDeployment {
	return &TagDeployment{
		deploymentRepo: deploymentRepo,
		resolver:       resolver,
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
	// Use the deployment resolver
	query := domain.DeploymentQuery{
		Reference: params.Identifier,
		ChainID:   params.ChainID,
		Namespace: params.Namespace,
	}
	deployment, err := t.resolver.ResolveDeployment(ctx, query)
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
