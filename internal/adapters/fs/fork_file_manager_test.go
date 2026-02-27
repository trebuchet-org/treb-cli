package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

func newTestForkFileManager(t *testing.T) (*ForkFileManagerAdapter, string) {
	t.Helper()
	tmpDir := t.TempDir()
	cfg := &config.RuntimeConfig{
		DataDir: tmpDir,
	}
	return NewForkFileManagerAdapter(cfg), tmpDir
}

// writeTestFile creates a file in the data directory with the given content.
func writeTestFile(t *testing.T, dataDir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dataDir, name), []byte(content), 0644)
	require.NoError(t, err)
}

// readTestFile reads the content of a file in the data directory.
func readTestFile(t *testing.T, dataDir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dataDir, name))
	require.NoError(t, err)
	return string(data)
}

func TestForkFileManager_BackupAndRestore(t *testing.T) {
	mgr, dataDir := newTestForkFileManager(t)
	ctx := context.Background()

	// Create some registry files
	writeTestFile(t, dataDir, "deployments.json", `{"deployments": []}`)
	writeTestFile(t, dataDir, "transactions.json", `{"transactions": []}`)

	// Backup to snapshot 0
	err := mgr.BackupFiles(ctx, "sepolia", 0)
	require.NoError(t, err)

	// Verify backup files exist
	snapDir := mgr.snapshotDir("sepolia", 0)
	assert.FileExists(t, filepath.Join(snapDir, "deployments.json"))
	assert.FileExists(t, filepath.Join(snapDir, "transactions.json"))

	// Verify backup content matches
	backupData, err := os.ReadFile(filepath.Join(snapDir, "deployments.json"))
	require.NoError(t, err)
	assert.Equal(t, `{"deployments": []}`, string(backupData))

	// Modify the original files
	writeTestFile(t, dataDir, "deployments.json", `{"deployments": [{"id": "new"}]}`)

	// Restore from snapshot 0
	err = mgr.RestoreFiles(ctx, "sepolia", 0)
	require.NoError(t, err)

	// Verify restored content matches the backup
	restored := readTestFile(t, dataDir, "deployments.json")
	assert.Equal(t, `{"deployments": []}`, restored)
}

func TestForkFileManager_BackupSkipsMissingFiles(t *testing.T) {
	mgr, dataDir := newTestForkFileManager(t)
	ctx := context.Background()

	// Only create one file (others don't exist)
	writeTestFile(t, dataDir, "deployments.json", `{"deployments": []}`)

	// Backup should succeed, skipping missing files
	err := mgr.BackupFiles(ctx, "sepolia", 0)
	require.NoError(t, err)

	// Only the existing file should be backed up
	snapDir := mgr.snapshotDir("sepolia", 0)
	assert.FileExists(t, filepath.Join(snapDir, "deployments.json"))
	assert.NoFileExists(t, filepath.Join(snapDir, "transactions.json"))
	assert.NoFileExists(t, filepath.Join(snapDir, "safe-txs.json"))
	assert.NoFileExists(t, filepath.Join(snapDir, "registry.json"))
	assert.NoFileExists(t, filepath.Join(snapDir, "addressbook.json"))
}

func TestForkFileManager_RestoreSkipsMissingFiles(t *testing.T) {
	mgr, dataDir := newTestForkFileManager(t)
	ctx := context.Background()

	// Create a backup with only one file
	writeTestFile(t, dataDir, "deployments.json", `{"deployments": []}`)
	err := mgr.BackupFiles(ctx, "sepolia", 0)
	require.NoError(t, err)

	// Create an extra file in .treb/ that wasn't in the backup
	writeTestFile(t, dataDir, "transactions.json", `{"transactions": ["extra"]}`)

	// Restore should succeed, only overwriting files that exist in backup
	err = mgr.RestoreFiles(ctx, "sepolia", 0)
	require.NoError(t, err)

	// The backed-up file should be restored
	restored := readTestFile(t, dataDir, "deployments.json")
	assert.Equal(t, `{"deployments": []}`, restored)

	// The file that wasn't in the backup should still have its current content
	extra := readTestFile(t, dataDir, "transactions.json")
	assert.Equal(t, `{"transactions": ["extra"]}`, extra)
}

func TestForkFileManager_BackupAllFiles(t *testing.T) {
	mgr, dataDir := newTestForkFileManager(t)
	ctx := context.Background()

	// Create all registry files
	for _, name := range registryFiles {
		writeTestFile(t, dataDir, name, `{"file": "`+name+`"}`)
	}

	err := mgr.BackupFiles(ctx, "mainnet", 0)
	require.NoError(t, err)

	// Verify all files backed up
	snapDir := mgr.snapshotDir("mainnet", 0)
	for _, name := range registryFiles {
		assert.FileExists(t, filepath.Join(snapDir, name))
		data, err := os.ReadFile(filepath.Join(snapDir, name))
		require.NoError(t, err)
		assert.Equal(t, `{"file": "`+name+`"}`, string(data))
	}
}

func TestForkFileManager_MultipleSnapshots(t *testing.T) {
	mgr, dataDir := newTestForkFileManager(t)
	ctx := context.Background()

	// Snapshot 0: initial state
	writeTestFile(t, dataDir, "deployments.json", `{"version": 0}`)
	err := mgr.BackupFiles(ctx, "sepolia", 0)
	require.NoError(t, err)

	// Snapshot 1: after first deploy
	writeTestFile(t, dataDir, "deployments.json", `{"version": 1}`)
	err = mgr.BackupFiles(ctx, "sepolia", 1)
	require.NoError(t, err)

	// Snapshot 2: after second deploy
	writeTestFile(t, dataDir, "deployments.json", `{"version": 2}`)
	err = mgr.BackupFiles(ctx, "sepolia", 2)
	require.NoError(t, err)

	// Verify each snapshot has distinct content
	data0, err := os.ReadFile(filepath.Join(mgr.snapshotDir("sepolia", 0), "deployments.json"))
	require.NoError(t, err)
	assert.Equal(t, `{"version": 0}`, string(data0))

	data1, err := os.ReadFile(filepath.Join(mgr.snapshotDir("sepolia", 1), "deployments.json"))
	require.NoError(t, err)
	assert.Equal(t, `{"version": 1}`, string(data1))

	data2, err := os.ReadFile(filepath.Join(mgr.snapshotDir("sepolia", 2), "deployments.json"))
	require.NoError(t, err)
	assert.Equal(t, `{"version": 2}`, string(data2))

	// Restore from snapshot 1 (middle)
	err = mgr.RestoreFiles(ctx, "sepolia", 1)
	require.NoError(t, err)
	restored := readTestFile(t, dataDir, "deployments.json")
	assert.Equal(t, `{"version": 1}`, restored)
}

func TestForkFileManager_CleanupForkDir(t *testing.T) {
	mgr, dataDir := newTestForkFileManager(t)
	ctx := context.Background()

	// Create some snapshots
	writeTestFile(t, dataDir, "deployments.json", `{"test": true}`)
	err := mgr.BackupFiles(ctx, "sepolia", 0)
	require.NoError(t, err)
	err = mgr.BackupFiles(ctx, "sepolia", 1)
	require.NoError(t, err)

	// Verify the fork dir exists
	forkDir := filepath.Join(dataDir, "priv", "fork", "sepolia")
	_, err = os.Stat(forkDir)
	require.NoError(t, err)

	// Cleanup
	err = mgr.CleanupForkDir(ctx, "sepolia")
	require.NoError(t, err)

	// Fork dir should be gone
	_, err = os.Stat(forkDir)
	assert.True(t, os.IsNotExist(err))
}

func TestForkFileManager_CleanupNonExistentDir(t *testing.T) {
	mgr, _ := newTestForkFileManager(t)
	ctx := context.Background()

	// Cleaning up a non-existent dir should not error
	err := mgr.CleanupForkDir(ctx, "nonexistent")
	require.NoError(t, err)
}

func TestForkFileManager_BackupCreatesDirectoryTree(t *testing.T) {
	mgr, dataDir := newTestForkFileManager(t)
	ctx := context.Background()

	writeTestFile(t, dataDir, "deployments.json", `{}`)

	err := mgr.BackupFiles(ctx, "sepolia", 0)
	require.NoError(t, err)

	// Verify the full directory tree was created
	snapDir := mgr.snapshotDir("sepolia", 0)
	info, err := os.Stat(snapDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestForkFileManager_BackupNoFiles(t *testing.T) {
	mgr, _ := newTestForkFileManager(t)
	ctx := context.Background()

	// No registry files exist at all - should succeed
	err := mgr.BackupFiles(ctx, "sepolia", 0)
	require.NoError(t, err)

	// Snapshot directory should be created even if empty
	snapDir := mgr.snapshotDir("sepolia", 0)
	info, err := os.Stat(snapDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestForkFileManager_CleanupDoesNotAffectOtherNetworks(t *testing.T) {
	mgr, dataDir := newTestForkFileManager(t)
	ctx := context.Background()

	writeTestFile(t, dataDir, "deployments.json", `{}`)

	// Create snapshots for two networks
	err := mgr.BackupFiles(ctx, "sepolia", 0)
	require.NoError(t, err)
	err = mgr.BackupFiles(ctx, "mainnet", 0)
	require.NoError(t, err)

	// Cleanup only sepolia
	err = mgr.CleanupForkDir(ctx, "sepolia")
	require.NoError(t, err)

	// Sepolia should be gone
	_, err = os.Stat(filepath.Join(dataDir, "priv", "fork", "sepolia"))
	assert.True(t, os.IsNotExist(err))

	// Mainnet should still exist
	_, err = os.Stat(filepath.Join(dataDir, "priv", "fork", "mainnet"))
	require.NoError(t, err)
}
