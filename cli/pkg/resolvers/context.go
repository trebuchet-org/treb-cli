package resolvers

import "github.com/trebuchet-org/treb-cli/cli/pkg/types"

// Context holds the resolver configuration and state
type ContractsResolver struct {
	lookup      types.ContractLookup
	interactive bool
}

type DeploymentsResolver struct {
	lookup      types.DeploymentLookup
	interactive bool
}

func NewContractsResolver(lookup types.ContractLookup, interactive bool) *ContractsResolver {
	return &ContractsResolver{
		lookup:      lookup,
		interactive: interactive,
	}
}

func NewDeploymentsResolver(lookup types.DeploymentLookup, interactive bool) *DeploymentsResolver {
	return &DeploymentsResolver{
		lookup:      lookup,
		interactive: interactive,
	}
}
