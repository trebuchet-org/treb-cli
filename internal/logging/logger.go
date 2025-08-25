package logging

import (
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/google/wire"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// Logger wraps slog.Logger with additional context
type Logger struct {
	*slog.Logger
}

var LoggingSet = wire.NewSet(
	NewLogger,
)

// NewLogger creates a new logger based on runtime configuration
func NewLogger(cfg *config.RuntimeConfig) *slog.Logger {
	level := slog.LevelInfo

	if val := strings.ToLower(os.Getenv("TREB_LOG_LEVEL")); val != "" {
		switch val {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			// unknown value, keep default
		}
	}

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove time in non-debug mode for cleaner output
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			// Shorten source paths
			if a.Key == slog.SourceKey {
				source := a.Value.Any().(*slog.Source)
				source.File = shortPath(source.File)
			}
			return a
		},
	}

	// if cfg.Debug {
	// 	opts.Level = slog.LevelDebug
	// 	opts.AddSource = true
	// }

	var handler slog.Handler = slog.NewTextHandler(os.Stderr, opts)

	return slog.New(handler)
}

// shortPath returns a shortened version of the file path
func shortPath(file string) string {
	// Try to make paths relative to project root
	if idx := strings.Index(file, "treb-cli/"); idx != -1 {
		return file[idx+9:] // Skip "treb-cli/"
	}
	// Otherwise, just return the file name
	_, f, _, _ := runtime.Caller(0)
	if idx := strings.LastIndex(f, "/"); idx != -1 {
		if idx2 := strings.LastIndex(file, f[:idx]); idx2 != -1 {
			return file[idx2+len(f[:idx])+1:]
		}
	}
	// Last resort: just the filename
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return file
}
