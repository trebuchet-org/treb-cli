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

// Ensure NopSink implements ProgressSink
var _ usecase.ProgressSink = (*NopSink)(nil)