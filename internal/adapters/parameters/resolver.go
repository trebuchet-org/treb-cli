package parameters

import (
	"context"
	"fmt"
	"strings"

	pkgconfig "github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parameters"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// ParameterResolverAdapter adapts the existing parameter resolver
type ParameterResolverAdapter struct {
	workDir     string
	trebConfig  *pkgconfig.TrebConfig
	namespace   string
	network     string
	chainID     uint64
	interactive bool
}

// NewParameterResolverAdapter creates a new parameter resolver adapter
func NewParameterResolverAdapter(cfg *config.RuntimeConfig) *ParameterResolverAdapter {
	// Get chain ID from network config if available
	var chainID uint64
	if cfg.Network != nil {
		chainID = cfg.Network.ChainID
	}

	// Convert RuntimeConfig TrebConfig to v1 TrebConfig
	var v1TrebConfig *pkgconfig.TrebConfig
	if cfg.TrebConfig != nil {
		v1TrebConfig = &pkgconfig.TrebConfig{
			Senders:         make(map[string]pkgconfig.SenderConfig),
			LibraryDeployer: cfg.TrebConfig.LibraryDeployer,
		}
		// Convert senders from cfg.TrebConfig.Senders
		for name, sender := range cfg.TrebConfig.Senders {
			v1Sender := pkgconfig.SenderConfig{
				Type:           sender.Type,
				Address:        sender.Account,
				PrivateKey:     sender.PrivateKey,
				Safe:           sender.Safe,
				DerivationPath: sender.DerivationPath,
			}
			// Note: v1 doesn't have Proposer config, it uses Signer instead
			if sender.Proposer != nil && sender.Type == "safe" {
				// For safe senders, set the signer based on proposer
				if sender.Proposer.Type == "private_key" {
					v1Sender.Signer = "private_key"
				} else if sender.Proposer.Type == "ledger" {
					v1Sender.Signer = "ledger"
				}
			}
			v1TrebConfig.Senders[name] = v1Sender
		}
	}

	return &ParameterResolverAdapter{
		workDir:     cfg.ProjectRoot,
		trebConfig:  v1TrebConfig,
		namespace:   cfg.Namespace,
		network:     getNetworkName(cfg),
		chainID:     chainID,
		interactive: !cfg.NonInteractive,
	}
}

// getNetworkName extracts the network name from config
func getNetworkName(cfg *config.RuntimeConfig) string {
	if cfg.Network != nil {
		return cfg.Network.Name
	}
	return "local"
}

// ResolveParameters resolves parameter values from various sources
func (a *ParameterResolverAdapter) ResolveParameters(
	ctx context.Context,
	params []domain.ScriptParameter,
	values map[string]string,
) (map[string]string, error) {
	// Convert domain parameters to v1 parameters
	v1Params := make([]parameters.Parameter, len(params))
	for i, p := range params {
		v1Params[i] = parameters.Parameter{
			Name:        p.Name,
			Type:        mapDomainToV1Type(p.Type),
			Description: p.Description,
			Optional:    p.Optional,
		}
	}

	// Create v1 parameter resolver
	resolver, err := parameters.NewParameterResolver(
		a.workDir,
		a.trebConfig,
		a.namespace,
		a.network,
		a.chainID,
		a.interactive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create parameter resolver: %w", err)
	}

	// Resolve using v1 resolver
	resolved, err := resolver.ResolveAll(v1Params, values)
	if err != nil {
		// Don't fail here if interactive, as we might prompt later
		if !a.interactive {
			return nil, err
		}
	}

	return resolved, nil
}

// ValidateParameters validates that all required parameters have values
func (a *ParameterResolverAdapter) ValidateParameters(
	ctx context.Context,
	params []domain.ScriptParameter,
	values map[string]string,
) error {
	var missing []string
	for _, param := range params {
		if !param.Optional && values[param.Name] == "" {
			missing = append(missing, param.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required parameters: %s", strings.Join(missing, ", "))
	}

	return nil
}

// ParameterPrompterAdapter adapts the existing parameter prompter
type ParameterPrompterAdapter struct {
	resolver *ParameterResolverAdapter
}

// NewParameterPrompterAdapter creates a new parameter prompter adapter
func NewParameterPrompterAdapter(resolver *ParameterResolverAdapter) *ParameterPrompterAdapter {
	return &ParameterPrompterAdapter{
		resolver: resolver,
	}
}

// PromptForParameters prompts the user for missing parameter values
func (p *ParameterPrompterAdapter) PromptForParameters(
	ctx context.Context,
	params []domain.ScriptParameter,
	existing map[string]string,
) (map[string]string, error) {
	// Convert domain parameters to v1 parameters
	v1Params := make([]parameters.Parameter, len(params))
	for i, param := range params {
		v1Params[i] = parameters.Parameter{
			Name:        param.Name,
			Type:        mapDomainToV1Type(param.Type),
			Description: param.Description,
			Optional:    param.Optional,
		}
	}

	// Create v1 parameter resolver for the prompter
	resolver, err := parameters.NewParameterResolver(
		p.resolver.workDir,
		p.resolver.trebConfig,
		p.resolver.namespace,
		p.resolver.network,
		p.resolver.chainID,
		true, // Always interactive for prompter
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create parameter resolver: %w", err)
	}

	// Create prompter
	prompter := parameters.NewParameterPrompter(resolver)

	// Prompt for missing parameters
	result, err := prompter.PromptForMissingParameters(v1Params, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for parameters: %w", err)
	}

	// Re-resolve with prompted values
	return resolver.ResolveAll(v1Params, result)
}

// mapDomainToV1Type maps domain parameter types to v1 parameter types
func mapDomainToV1Type(domainType domain.ParameterType) parameters.ParameterType {
	switch domainType {
	case domain.ParamTypeString:
		return parameters.TypeString
	case domain.ParamTypeAddress:
		return parameters.TypeAddress
	case domain.ParamTypeUint256:
		return parameters.TypeUint256
	case domain.ParamTypeInt256:
		return parameters.TypeInt256
	case domain.ParamTypeBytes32:
		return parameters.TypeBytes32
	case domain.ParamTypeBytes:
		return parameters.TypeBytes
	case domain.ParamTypeBool:
		return parameters.TypeBool
	case domain.ParamTypeSender:
		return parameters.TypeSender
	case domain.ParamTypeDeployment:
		return parameters.TypeDeployment
	case domain.ParamTypeArtifact:
		return parameters.TypeArtifact
	default:
		return parameters.TypeString
	}
}