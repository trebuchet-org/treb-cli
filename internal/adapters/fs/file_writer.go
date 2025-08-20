package fs

import (
	"context"
	"os"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// FileWriterAdapter handles file system operations for scripts
type FileWriterAdapter struct {
	// No state needed for now
}

// NewFileWriterAdapter creates a new file writer adapter
func NewFileWriterAdapter(cfg *config.RuntimeConfig) (*FileWriterAdapter, error) {
	return &FileWriterAdapter{}, nil
}

// WriteScript writes content to a file
func (f *FileWriterAdapter) WriteScript(ctx context.Context, path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// FileExists checks if a file exists
func (f *FileWriterAdapter) FileExists(ctx context.Context, path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// EnsureDirectory ensures a directory exists
func (f *FileWriterAdapter) EnsureDirectory(ctx context.Context, path string) error {
	return os.MkdirAll(path, 0755)
}

// Ensure the adapter implements the interface
var _ usecase.FileWriter = (*FileWriterAdapter)(nil)
