package fs

import "github.com/trebuchet-org/treb-cli/internal/domain"

// matchSafeTransactionStatus checks if safe transaction status matches the filter
func matchSafeTransactionStatus(safeStatus domain.SafeTxStatus, filterStatus domain.TransactionStatus) bool {
	// Map safe transaction status to transaction status for comparison
	switch safeStatus {
	case domain.SafeTxStatusQueued:
		return filterStatus == domain.TransactionStatusQueued
	case domain.SafeTxStatusExecuted:
		return filterStatus == domain.TransactionStatusExecuted
	case domain.SafeTxStatusFailed:
		return filterStatus == domain.TransactionStatusFailed
	default:
		return false
	}
}