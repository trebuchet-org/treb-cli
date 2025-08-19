package fs

import "github.com/trebuchet-org/treb-cli/internal/domain/models"

// matchSafeTransactionStatus checks if safe transaction status matches the filter
func matchSafeTransactionStatus(safeStatus models.SafeTxStatus, filterStatus models.TransactionStatus) bool {
	// Map safe transaction status to transaction status for comparison
	switch safeStatus {
	case models.SafeTxStatusQueued:
		return filterStatus == models.TransactionStatusQueued
	case models.SafeTxStatusExecuted:
		return filterStatus == models.TransactionStatusExecuted
	case models.SafeTxStatusFailed:
		return filterStatus == models.TransactionStatusFailed
	default:
		return false
	}
}

