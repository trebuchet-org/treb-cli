package script

import (
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
)

// contractInfoAdapter wraps contracts.ContractInfo to implement abi.ContractInfo
type contractInfoAdapter struct {
	info *contracts.ContractInfo
}

func (a *contractInfoAdapter) GetArtifactPath() string {
	if a.info == nil {
		return ""
	}
	return a.info.ArtifactPath
}

// indexerAdapter wraps contracts.Indexer to implement abi.ContractLookup
type indexerAdapter struct {
	indexer *contracts.Indexer
}

// GetContractByArtifact adapts the concrete type to the interface
func (a *indexerAdapter) GetContractByArtifact(artifact string) abi.ContractInfo {
	info := a.indexer.GetContractByArtifact(artifact)
	if info == nil {
		return nil
	}
	return &contractInfoAdapter{info: info}
}