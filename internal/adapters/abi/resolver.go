package abi

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/bindings"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

type ABIResolver struct {
	config         *config.RuntimeConfig
	contracts      usecase.ContractRepository
	deploymentRepo usecase.DeploymentRepository
	execution      *forge.HydratedRunResult
}

func NewABIResolver(config *config.RuntimeConfig, contracts usecase.ContractRepository, deploymentRepo usecase.DeploymentRepository) *ABIResolver {
	return &ABIResolver{
		config:         config,
		contracts:      contracts,
		deploymentRepo: deploymentRepo,
	}
}

// XXX: Need a better way to do this.
func (r *ABIResolver) SetExecution(execution *forge.HydratedRunResult) {
	r.execution = execution
}

func (r *ABIResolver) Get(ctx context.Context, artifact *models.Artifact) (*abi.ABI, error) {
	abi, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	return &abi, err
}

func (r *ABIResolver) FindByRef(ctx context.Context, contractRef string) (*abi.ABI, error) {
	query := domain.ContractQuery{Query: &contractRef}
	// Use FindContracts (index-only, no build-on-miss) since ABI resolution
	// happens after forge script has already compiled everything. Triggering
	// a build here would cause a 1-2 minute hang during output rendering.
	contracts, err := r.contracts.FindContracts(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search contracts: %w", err)
	}
	if len(contracts) == 0 {
		return nil, domain.NoContractsMatchErr{Query: query}
	}

	if len(contracts) > 1 {
		return nil, domain.AmbiguousFilterErr{Query: query}
	}

	if contracts[0].Artifact == nil {
		return nil, domain.MissingArtifactErr{Contract: contracts[0]}
	}

	return r.Get(ctx, contracts[0].Artifact)
}

func (r *ABIResolver) FindByAddress(ctx context.Context, address common.Address) (*abi.ABI, error) {
	// Check for well-known contracts first
	if address == common.HexToAddress("0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed") {
		// CreateX factory address
		createXABI, err := abi.JSON(strings.NewReader(bindings.CreateXMetaData.ABI))
		if err != nil {
			return nil, err
		}
		return &createXABI, nil
	}

	if r.execution != nil {
		for _, deployment := range r.execution.Deployments {
			if deployment.Address == address {
				return r.FindByRef(ctx, deployment.Event.Artifact)
			}
		}
	}

	deployment, err := r.deploymentRepo.GetDeploymentByAddress(ctx, r.config.Network.ChainID, address.String())
	if err != nil {
		return nil, err
	}
	if deployment == nil {
		return nil, domain.NoDeploymentErr{ChainID: r.config.Network.ChainID, Address: address}
	}

	abi, err := r.FindByRef(ctx, deployment.Artifact.Path)
	if err != nil {
		return nil, err
	}

	if r.execution != nil {
		if proxyRel, exists := r.execution.ProxyRelationships[address]; exists {
			if implAbi, err := r.FindByAddress(ctx, proxyRel.ImplementationAddress); err != nil {
				return nil, err
			} else if implAbi != nil {
				maps.Copy(abi.Methods, implAbi.Methods)
			}
		}
	}

	return abi, nil
}

var _ usecase.ABIResolver = (&ABIResolver{})
