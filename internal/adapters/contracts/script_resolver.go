package contracts

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/resolvers"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parameters"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// ScriptResolverAdapter adapts the existing contracts resolver to the ScriptResolver interface
type ScriptResolverAdapter struct {
	indexer     *contracts.Indexer
	resolver    *resolvers.ContractsResolver
	interactive bool
}

// NewScriptResolverAdapter creates a new script resolver adapter
func NewScriptResolverAdapter(cfg *config.RuntimeConfig) (*ScriptResolverAdapter, error) {
	indexer, err := contracts.GetGlobalIndexer(cfg.ProjectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize contract indexer: %w", err)
	}

	resolver := resolvers.NewContractsResolver(indexer, !cfg.NonInteractive)

	return &ScriptResolverAdapter{
		indexer:     indexer,
		resolver:    resolver,
		interactive: !cfg.NonInteractive,
	}, nil
}

// ResolveScript resolves a script path or name to script info
func (a *ScriptResolverAdapter) ResolveScript(ctx context.Context, pathOrName string) (*domain.ScriptInfo, error) {
	// Use the existing resolver to find the script contract
	contractInfo, err := a.resolver.ResolveContract(pathOrName, types.ScriptContractFilter())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve script contract: %w", err)
	}

	// Convert to domain script info
	scriptInfo := &domain.ScriptInfo{
		Path:         contractInfo.Path,
		Name:         contractInfo.Name,
		ContractName: contractInfo.Name,
	}

	// Convert artifact if available
	if contractInfo.Artifact != nil {
		scriptInfo.Artifact = &domain.ContractArtifact{
			// Map fields as needed
		}
	}

	return scriptInfo, nil
}

// GetScriptParameters extracts parameters from a script's artifact
func (a *ScriptResolverAdapter) GetScriptParameters(ctx context.Context, script *domain.ScriptInfo) ([]domain.ScriptParameter, error) {
	// Get the contract info from indexer
	contractInfo, err := a.indexer.GetContract(script.Path)
	if err != nil || contractInfo == nil || contractInfo.Artifact == nil {
		return []domain.ScriptParameter{}, nil
	}

	// Use the existing parameter parser
	parser := parameters.NewParameterParser()
	params, err := parser.ParseFromArtifact(contractInfo.Artifact)
	if err != nil {
		return nil, fmt.Errorf("failed to parse script parameters: %w", err)
	}

	// Convert to domain parameters
	var domainParams []domain.ScriptParameter
	for _, p := range params {
		domainParams = append(domainParams, domain.ScriptParameter{
			Name:        p.Name,
			Type:        mapParameterType(p.Type),
			Description: p.Description,
			Optional:    p.Optional,
		})
	}

	return domainParams, nil
}

// mapParameterType maps v1 parameter types to domain parameter types
func mapParameterType(v1Type parameters.ParameterType) domain.ParameterType {
	switch v1Type {
	case parameters.TypeString:
		return domain.ParamTypeString
	case parameters.TypeAddress:
		return domain.ParamTypeAddress
	case parameters.TypeUint256:
		return domain.ParamTypeUint256
	case parameters.TypeInt256:
		return domain.ParamTypeInt256
	case parameters.TypeBytes32:
		return domain.ParamTypeBytes32
	case parameters.TypeBytes:
		return domain.ParamTypeBytes
	case parameters.TypeBool:
		return domain.ParamTypeBool
	case parameters.TypeSender:
		return domain.ParamTypeSender
	case parameters.TypeDeployment:
		return domain.ParamTypeDeployment
	case parameters.TypeArtifact:
		return domain.ParamTypeArtifact
	default:
		return domain.ParamTypeString
	}
}