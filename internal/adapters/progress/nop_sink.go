package progress

import (
	"context"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NopSink is a no-op implementation of ProgressSink
type NopSink struct{}

// NewNopSink creates a new no-op progress sink
func NewNopSink() usecase.ProgressSink {
	return &NopSink{}
}

// OnProgress does nothing with progress events
func (n *NopSink) OnProgress(ctx context.Context, event usecase.ProgressEvent) {
	// No-op
}

// Info does nothing with info messages
func (n *NopSink) Info(message string) {
	// No-op
}

// Error does nothing with error messages
func (n *NopSink) Error(message string) {
	// No-op
}

// Ensure NopSink implements ProgressSink
var _ usecase.ProgressSink = (*NopSink)(nil)