package parser

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ExecutionParserAdapter adapts the internal parser to the ExecutionParser interface
type ExecutionParserAdapter struct {
	parser *InternalParser
}

// NewExecutionParserAdapter creates a new execution parser adapter
func NewExecutionParserAdapter(projectRoot string) *ExecutionParserAdapter {
	return &ExecutionParserAdapter{
		parser: NewInternalParser(projectRoot),
	}
}

// ParseExecution parses the script output into a structured execution result
func (a *ExecutionParserAdapter) ParseExecution(
	ctx context.Context,
	output *usecase.ScriptExecutionOutput,
	network string,
	chainID uint64,
) (*domain.ScriptExecution, error) {
	return a.parser.ParseExecution(ctx, output, network, chainID)
}

// EnrichFromBroadcast enriches execution data from broadcast files
func (a *ExecutionParserAdapter) EnrichFromBroadcast(
	ctx context.Context,
	execution *domain.ScriptExecution,
	broadcastPath string,
) error {
	// The internal parser handles broadcast parsing as part of ParseExecution
	// This is a no-op for compatibility
	return nil
}