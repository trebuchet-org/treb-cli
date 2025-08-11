package domain

import "errors"

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