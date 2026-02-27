package fs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// registryFiles is the list of .treb/ files to backup and restore during fork mode.
var registryFiles = []string{
	"deployments.json",
	"transactions.json",
	"safe-txs.json",
	"registry.json",
	"addressbook.json",
}

// ForkFileManagerAdapter implements ForkFileManager using the file system
type ForkFileManagerAdapter struct {
	dataDir string
}

// NewForkFileManagerAdapter creates a new ForkFileManagerAdapter
func NewForkFileManagerAdapter(cfg *config.RuntimeConfig) *ForkFileManagerAdapter {
	return &ForkFileManagerAdapter{
		dataDir: cfg.DataDir,
	}
}

// snapshotDir returns the path to a snapshot directory for a given network and index.
func (m *ForkFileManagerAdapter) snapshotDir(network string, snapshotIndex int) string {
	return filepath.Join(m.dataDir, "priv", "fork", network, "snapshots", fmt.Sprintf("%d", snapshotIndex))
}

// BackupFiles copies registry files from .treb/ to the snapshot directory.
func (m *ForkFileManagerAdapter) BackupFiles(_ context.Context, network string, snapshotIndex int) error {
	destDir := m.snapshotDir(network, snapshotIndex)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	for _, name := range registryFiles {
		src := filepath.Join(m.dataDir, name)
		dst := filepath.Join(destDir, name)
		if err := copyFile(src, dst); err != nil {
			if os.IsNotExist(err) {
				continue // skip files that don't exist
			}
			return fmt.Errorf("failed to backup %s: %w", name, err)
		}
	}

	return nil
}

// RestoreFiles copies registry files from the snapshot directory back to .treb/.
func (m *ForkFileManagerAdapter) RestoreFiles(_ context.Context, network string, snapshotIndex int) error {
	srcDir := m.snapshotDir(network, snapshotIndex)

	for _, name := range registryFiles {
		src := filepath.Join(srcDir, name)
		dst := filepath.Join(m.dataDir, name)
		if err := copyFile(src, dst); err != nil {
			if os.IsNotExist(err) {
				continue // skip files that don't exist in backup
			}
			return fmt.Errorf("failed to restore %s: %w", name, err)
		}
	}

	return nil
}

// CleanupForkDir removes the entire fork directory for a network.
func (m *ForkFileManagerAdapter) CleanupForkDir(_ context.Context, network string) error {
	forkDir := filepath.Join(m.dataDir, "priv", "fork", network)
	err := os.RemoveAll(forkDir)
	if err != nil {
		return fmt.Errorf("failed to cleanup fork directory: %w", err)
	}
	return nil
}

// copyFile copies a single file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(filepath.Clean(dst), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode()) //nolint:gosec // paths are constructed internally, not from user input
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// Ensure ForkFileManagerAdapter implements ForkFileManager
var _ usecase.ForkFileManager = (*ForkFileManagerAdapter)(nil)
