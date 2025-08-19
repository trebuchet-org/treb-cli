package abi

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

type ABIResolver struct {
	config      *config.RuntimeConfig
	contracts   usecase.ContractIndexer
	deployments usecase.DeploymentStore
}

func NewABIResolver(config *config.RuntimeConfig, contracts usecase.ContractIndexer, deployments usecase.DeploymentStore) *ABIResolver {
	return &ABIResolver{
		config:      config,
		contracts:   contracts,
		deployments: deployments,
	}
}

func (r *ABIResolver) Get(ctx context.Context, artifact *models.Artifact) (*abi.ABI, error) {
	abi, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	return &abi, err
}

func (r *ABIResolver) FindByRef(ctx context.Context, contractRef string) (*abi.ABI, error) {
	query := domain.ContractQuery{Query: &contractRef}
	contracts := r.contracts.SearchContracts(ctx, query)
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
	deployment, err := r.deployments.GetDeploymentByAddress(ctx, r.config.Network.ChainID, address.String())
	if err != nil {
		return nil, err
	}
	if deployment == nil {
		return nil, domain.NoDeploymentErr{ChainID: r.config.Network.ChainID, Address: address}
	}

	return r.FindByRef(ctx, deployment.Artifact.Path)

}

var _ usecase.ABIResolver = (&ABIResolver{})
