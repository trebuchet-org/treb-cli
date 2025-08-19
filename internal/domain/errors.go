package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// Sentinel errors for domain operations
var (
	// ErrNotFound is returned when a requested resource doesn't exist
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists is returned when trying to create a resource that already exists
	ErrAlreadyExists = errors.New("already exists")

	// ErrInvalidAddress is returned when an Ethereum address is invalid
	ErrInvalidAddress = errors.New("invalid address")

	// ErrInvalidChainID is returned when a chain ID is invalid
	ErrInvalidChainID = errors.New("invalid chain ID")

	// ErrInvalidDeployment is returned when deployment data is invalid
	ErrInvalidDeployment = errors.New("invalid deployment")

	// ErrNetworkMismatch is returned when network configurations don't match
	ErrNetworkMismatch = errors.New("network mismatch")

	// ErrContractNotFound is returned when a contract can't be found
	ErrContractNotFound = errors.New("contract not found")

	// ErrVerificationFailed is returned when contract verification fails
	ErrVerificationFailed = errors.New("verification failed")
)

type NoContractsMatchErr struct {
	Query ContractQuery
}

func (e NoContractsMatchErr) Error() string {
	return fmt.Sprintf("No contracts match query: %v", e.Query)
}

type AmbiguousFilterErr struct {
	Query   ContractQuery
	Matches []*models.Contract
}

func (e AmbiguousFilterErr) Error() string {
	// Sort contracts by artifact path for consistent output
	sortedContracts := make([]*models.Contract, len(e.Matches))
	copy(sortedContracts, e.Matches)

	sort.Slice(sortedContracts, func(i, j int) bool {
		// Sort by full artifact path (path:name)
		artifactI := fmt.Sprintf("%s:%s", sortedContracts[i].Path, sortedContracts[i].Name)
		artifactJ := fmt.Sprintf("%s:%s", sortedContracts[j].Path, sortedContracts[j].Name)
		return artifactI < artifactJ
	})

	var suggestions []string
	for _, contract := range sortedContracts {
		suggestion := fmt.Sprintf("  - %s (%s)", contract.Name, contract.Path)
		suggestions = append(suggestions, suggestion)
	}

	return fmt.Sprintf("multiple contracts found matching %v - use full path:contract format to disambiguate:\n%s",
		e.Query, strings.Join(suggestions, "\n"))
}

type MissingArtifactErr struct {
	Contract *models.Contract
}

func (e MissingArtifactErr) Error() string {
	return fmt.Sprintf("Missing Artifact for contract: %s:s", e.Contract.Path, e.Contract.Name)

}

type NoDeploymentErr struct {
	ChainID uint64
	Address common.Address
}

func (e NoDeploymentErr) Error() string {
	return fmt.Sprintf("No deployment registered at %s on chain %d", e.Address.String(), e.ChainID)
}
