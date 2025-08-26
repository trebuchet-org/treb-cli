package forge

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

type HydratedRunResult forge.HydratedRunResult

// GetDeploymentByAddress returns the deployment record for a given address
func (r *HydratedRunResult) GetDeploymentByAddress(address common.Address) *forge.Deployment {
	for _, dep := range r.Deployments {
		if dep.Address == address {
			return dep
		}
	}
	return nil
}

// GetProxyInfo returns proxy info for an address if it's a proxy
func (r *HydratedRunResult) GetProxyInfo(address common.Address) (*forge.ProxyInfo, bool) {
	rel, exists := r.ProxyRelationships[address]
	if !exists {
		return nil, false
	}

	info := &forge.ProxyInfo{
		Implementation: rel.ImplementationAddress,
		ProxyType:      string(rel.ProxyType),
		Admin:          rel.AdminAddress,
		Beacon:         rel.BeaconAddress,
	}

	return info, true
}

// GetTransactionByID returns a transaction by its ID
func (r *HydratedRunResult) GetTransactionByID(txID [32]byte) *forge.Transaction {
	for _, tx := range r.Transactions {
		if tx.TransactionId == txID {
			return tx
		}
	}
	return nil
}

// GetProxiesForImplementation returns all proxies pointing to an implementation
func (r *HydratedRunResult) GetProxiesForImplementation(implAddress common.Address) []*forge.ProxyRelationship {
	var proxies []*forge.ProxyRelationship
	for _, rel := range r.ProxyRelationships {
		if rel.ImplementationAddress == implAddress {
			proxies = append(proxies, rel)
		}
	}
	return proxies
}

var _ usecase.RunResultHydrator = (&RunResultHydrator{})
