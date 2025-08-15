package parser

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ExecutionParserAdapterNew is the new implementation that uses our internal parser
type ExecutionParserAdapterNew struct {
	parser *InternalParser
}

// NewExecutionParserAdapterNew creates a new execution parser adapter using internal implementation
func NewExecutionParserAdapterNew(cfg *config.RuntimeConfig) *ExecutionParserAdapterNew {
	return &ExecutionParserAdapterNew{
		parser: NewInternalParser(cfg.ProjectRoot),
	}
}

// ParseExecution parses the script output into a structured execution result
func (a *ExecutionParserAdapterNew) ParseExecution(
	ctx context.Context,
	output *usecase.ScriptExecutionOutput,
	network string,
	chainID uint64,
) (*domain.ScriptExecution, error) {
	return a.parser.ParseExecution(ctx, output, network, chainID)
}

// EnrichFromBroadcast enriches execution data from broadcast files
func (a *ExecutionParserAdapterNew) EnrichFromBroadcast(ctx context.Context, execution *domain.ScriptExecution, broadcastPath string) error {
	// This method would typically read and parse the broadcast file to enhance the execution data
	// For now, we'll just mark that broadcast data has been loaded
	if broadcastPath != "" {
		execution.BroadcastPath = broadcastPath
	}
	return nil
}

// Ensure it implements the interface
var _ usecase.ExecutionParser = (*ExecutionParserAdapterNew)(nil)