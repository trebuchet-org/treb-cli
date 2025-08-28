package helpers

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pelletier/go-toml/v2"
	"github.com/trebuchet-org/treb-cli/pkg/anvil"
)

// TestContext represents an isolated test environment
type TestContext struct {
	ID         string
	WorkDir    string
	AnvilNodes map[string]*AnvilNode
	// snapshotRef -> node -> snapshotId for reverting state
	Snapshots   map[string]map[string]string
	TrebContext *TrebContext
	inUse       bool
	mu          sync.Mutex
}

// TestContextPool manages a pool of test contexts for parallel execution
type TestContextPool struct {
	contexts    []*TestContext
	mu          sync.Mutex
	cond        *sync.Cond
	baseProject string
	poolSize    int
	setupOnce   sync.Once
	teardownCh  chan struct{}
}

type AnvilNode struct {
	*anvil.AnvilInstance
	URL string
}

var (
	globalPool *TestContextPool
)

// GetGlobalPool returns the global test context pool
func GetGlobalPool() *TestContextPool {
	return globalPool
}

// InitializeTestPool creates a global test context pool
func initializeTestPool(poolSize int) error {
	if globalPool != nil {
		return nil // Already initialized
	}

	baseProject := GetFixtureDir()

	pool := &TestContextPool{
		contexts:    make([]*TestContext, 0, poolSize),
		baseProject: baseProject,
		poolSize:    poolSize,
		teardownCh:  make(chan struct{}),
	}
	pool.cond = sync.NewCond(&pool.mu)

	// Build contracts once
	setupSpinner.UpdateMessage("Building contracts...")
	cmd := exec.Command("forge", "build")
	cmd.Dir = baseProject
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build contracts: %w", err)
	}

	// Create contexts in parallel
	setupSpinner.UpdateMessage(fmt.Sprintf("Creating %d test contexts...", poolSize))

	type contextResult struct {
		ctx *TestContext
		err error
		idx int
	}

	resultChan := make(chan contextResult, poolSize)

	// Launch goroutines to create contexts in parallel
	for i := range poolSize {
		go func(index int) {
			ctx, err := pool.createContext()
			resultChan <- contextResult{ctx: ctx, err: err, idx: index}
		}(i)
	}

	// Collect results
	results := make([]*TestContext, poolSize)
	successCount := 0

	for result := range resultChan {
		if result.err != nil {
			// Clean up already created contexts
			for _, c := range pool.contexts {
				if c != nil {
					c.Cleanup()
				}
			}
			return fmt.Errorf("failed to create context %d: %w", result.idx, result.err)
		}
		results[result.idx] = result.ctx
		successCount++
		setupSpinner.UpdateMessage(fmt.Sprintf("Initialized %d/%d test contexts", successCount, poolSize))
		if successCount == poolSize {
			close(resultChan)
		}
	}

	// Add all contexts to the pool
	pool.contexts = results
	globalPool = pool
	return nil
}

// getAvailablePort finds an available port
func getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port, nil
}

// createContext creates a new isolated test context
func (p *TestContextPool) createContext() (*TestContext, error) {
	ctx := &TestContext{
		ID:         uuid.New().String()[:8],
		WorkDir:    filepath.Join("/tmp", fmt.Sprintf("treb-test-workdir-%s", uuid.New().String()[:8])),
		AnvilNodes: make(map[string]*AnvilNode),
	}

	// Create lightweight workspace with symlinks
	if err := createLightweightWorkspace(p.baseProject, ctx.WorkDir); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Find available ports
	port1, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get port 1: %w", err)
	}
	port2, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get port 2: %w", err)
	}

	// Update foundry.toml with new ports
	foundryPath := filepath.Join(ctx.WorkDir, "foundry.toml")
	if err := updateFoundryConfig(foundryPath, port1, port2); err != nil {
		return nil, fmt.Errorf("failed to update foundry config: %w", err)
	}

	// Start anvil nodes
	node1Name := fmt.Sprintf("anvil-31337-%s", ctx.ID)
	node2Name := fmt.Sprintf("anvil-31338-%s", ctx.ID)

	// Start first node (quietly)
	if err := anvil.StartAnvilInstance(node1Name, fmt.Sprintf("%d", port1), "31337"); err != nil {
		return nil, fmt.Errorf("failed to start anvil node 1: %w", err)
	}
	ctx.AnvilNodes["anvil-31337"] = &AnvilNode{
		AnvilInstance: anvil.NewAnvilInstance(node1Name, fmt.Sprintf("%d", port1)).WithChainID("31337"),
		URL:           fmt.Sprintf("http://127.0.0.1:%d", port1),
	}

	// Start second node (quietly)
	if err := anvil.StartAnvilInstance(node2Name, fmt.Sprintf("%d", port2), "31338"); err != nil {
		anvil.StopAnvilInstance(node1Name, fmt.Sprintf("%d", port1))
		return nil, fmt.Errorf("failed to start anvil node 2: %w", err)
	}
	ctx.AnvilNodes["anvil-31338"] = &AnvilNode{
		AnvilInstance: anvil.NewAnvilInstance(node2Name, fmt.Sprintf("%d", port2)).WithChainID("31338"),
		URL:           fmt.Sprintf("http://127.0.0.1:%d", port2),
	}

	// Initialize snapshots map
	ctx.Snapshots = make(map[string]map[string]string)
	return ctx, nil
}

func (ctx *TestContext) TakeSnapshots(ref string) error {
	if _, exists := ctx.Snapshots[ref]; exists {
		return fmt.Errorf("Snapshot %s already exists", ref)
	}

	ctx.Snapshots[ref] = map[string]string{}

	for name, node := range ctx.AnvilNodes {
		if snapshotId, err := takeSnapshot(node.URL); err != nil {
			return fmt.Errorf("Failed to create snapshot: %v", err)
		} else {
			ctx.Snapshots[ref][name] = snapshotId
		}
	}

	return nil
}

func (ctx *TestContext) RevertSnapshots(ref string) error {
	var snapshot map[string]string
	var exists bool
	if snapshot, exists = ctx.Snapshots[ref]; !exists {
		return fmt.Errorf("Snapshot %s does not exist", ref)
	}

	for name, node := range ctx.AnvilNodes {
		if err := revertSnapshot(node.URL, snapshot[name]); err != nil {
			return fmt.Errorf("Failed to revert snapshot for node %s: %v\n", name, err)
		}
	}

	delete(ctx.Snapshots, ref)
	return nil
}

// updateFoundryConfig updates the foundry.toml with new ports
func updateFoundryConfig(path string, port1, port2 int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := toml.Unmarshal(data, &config); err != nil {
		return err
	}

	// Update RPC endpoints
	if rpcEndpoints, ok := config["rpc_endpoints"].(map[string]interface{}); ok {
		rpcEndpoints["anvil-31337"] = fmt.Sprintf("http://127.0.0.1:%d", port1)
		rpcEndpoints["anvil-31338"] = fmt.Sprintf("http://127.0.0.1:%d", port2)
	}

	// Write back
	data, err = toml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Acquire gets an available context from the pool
func (p *TestContextPool) Acquire(t *testing.T) *TestContext {
	t.Helper()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if all contexts are in use and skip cleanup is enabled
	availableCount := 0
	for _, ctx := range p.contexts {
		if !ctx.inUse {
			availableCount++
		}
	}

	if availableCount == 0 && ShouldSkipCleanup() {
		t.Fatal("No test contexts available and skip cleanup is enabled. " +
			"Either disable skip cleanup or increase parallelism to provide more contexts.")
	}

	for {
		for _, ctx := range p.contexts {
			if !ctx.inUse {
				ctx.inUse = true
				err := ctx.TakeSnapshots("%clean%")
				if err != nil {
					t.Fatal(err)
				}
				// Create TrebContext for the test
				ctx.TrebContext = NewTrebContext(t, ctx)
				// Don't change the global working directory in parallel tests
				// The TrebContext will use workDir to set cmd.Dir instead
				return ctx
			}
		}
		// Wait for a context to become available
		p.cond.Wait()
	}
}

// Release returns a context to the pool
func (p *TestContextPool) Release(ctx *TestContext) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Log work directory if skip cleanup is enabled
	if ShouldSkipCleanup() {
		ctx.TrebContext.t.Logf("🔍 Test work directory preserved at: %s", ctx.WorkDir)
	} else {
		if err := ctx.Clean(); err != nil {
			return err
		}
		ctx.inUse = false
		ctx.TrebContext = nil

		// Signal that a context is available
		p.cond.Signal()
	}

	return nil
}

func (ctx *TestContext) forgeClean() error {
	cmdCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	forgeClean := exec.CommandContext(cmdCtx, "forge", "clean")
	forgeClean.Dir = ctx.WorkDir
	return forgeClean.Run()
}

// Clean cleans up a test context for reuse
func (ctx *TestContext) Clean() error {
	// First, revert Anvil nodes to their initial snapshots
	// This ensures blockchain state is clean for the next test
	ctx.RevertSnapshots("%clean%")

	if err := ctx.forgeClean(); err != nil {
		return err
	}

	// Clean test artifacts (these are real directories we created)
	cleanDirs := []string{".treb", "broadcast", "cache", "out"}
	for _, dir := range cleanDirs {
		dirPath := filepath.Join(ctx.WorkDir, dir)
		os.RemoveAll(dirPath)
		// Recreate empty directories for next test
		os.MkdirAll(dirPath, 0755)
	}

	// Clean generated scripts except .gitkeep
	scriptDir := filepath.Join(ctx.WorkDir, "script", "deploy")
	if entries, err := os.ReadDir(scriptDir); err == nil {
		for _, entry := range entries {
			if entry.Name() != ".gitkeep" {
				os.Remove(filepath.Join(scriptDir, entry.Name()))
			}
		}
	}
	return nil
}

// Cleanup destroys the test context
func (ctx *TestContext) Cleanup() {
	// Stop anvil nodes
	for name, node := range ctx.AnvilNodes {
		anvil.StopAnvilInstance(node.Name, node.Port)
		delete(ctx.AnvilNodes, name)
	}

	// Remove work directory (unless skip cleanup is enabled)
	if !ShouldSkipCleanup() {
		os.RemoveAll(ctx.WorkDir)
	} else {
		fmt.Printf("🔍 Test context work directory preserved at: %s\n", ctx.WorkDir)
	}
}

// Shutdown cleans up all contexts
func (p *TestContextPool) Shutdown() {
	close(p.teardownCh)

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, ctx := range p.contexts {
		ctx.Cleanup()
	}
	p.contexts = nil
}

// createLightweightWorkspace creates a workspace with symlinks to most files
// and only copies what needs to be modified (foundry.toml and script directory)
func createLightweightWorkspace(src, dst string) error {
	// Create the workspace directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// List all items in the source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		switch entry.Name() {
		case ".treb":
			// Skip .treb directory
			continue

		case "foundry.toml":
			// Copy foundry.toml (we'll modify it later)
			if err := copyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy foundry.toml: %w", err)
			}

		case "script":
			// Copy script directory (for test isolation)
			if err := copyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy script directory: %w", err)
			}

		case "src":
			// Copy src directory to avoid Foundry symlink resolution issues
			if err := copyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy src directory: %w", err)
			}
		case ".env.example":
			if err := mergeFile(srcPath, filepath.Join(dst, ".env"), "\n\n"); err != nil {
				return fmt.Errorf("failed to copy .env file: %w", err)
			}
		case ".env":
			if err := mergeFile(srcPath, filepath.Join(dst, ".env"), "\n\n"); err != nil {
				return fmt.Errorf("failed to copy .env file: %w", err)
			}
		default:
			// Skip directories that should be created empty
			testDirs := []string{".treb", "broadcast", "cache", "out"}
			if slices.Contains(testDirs, entry.Name()) {
				// These will be created empty below, skip them here
				continue
			}

			// Symlink everything else (src, lib, test, etc.)
			if err := os.Symlink(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to create symlink for %s: %w", entry.Name(), err)
			}
		}
	}

	// Create directories that tests might need (empty)
	testDirs := []string{".treb", "broadcast", "cache", "out"}
	for _, dir := range testDirs {
		if err := os.MkdirAll(filepath.Join(dst, dir), 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Copy file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// mergeFile merges a file to dst if already exists
func mergeFile(src, dst, separator string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	dstFile.WriteString(separator)

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		// Skip git directory
		if strings.HasPrefix(relPath, ".git") {
			return nil
		}

		// Handle symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkTarget, dstPath)
		}

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return err
		}

		return os.Chmod(dstPath, info.Mode())
	})
}

// takeSnapshot takes a snapshot of the current blockchain state
func takeSnapshot(rpcURL string) (string, error) {
	cmd := exec.Command("cast", "rpc", "--rpc-url", rpcURL, "evm_snapshot")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to take snapshot: %w", err)
	}

	// The output is JSON, extract the snapshot ID
	snapshotID := strings.Trim(string(output), "\"\n")
	return snapshotID, nil
}

// revertSnapshot reverts the blockchain to a previous snapshot
func revertSnapshot(rpcURL, snapshotID string) error {
	cmd := exec.Command("cast", "rpc", "--rpc-url", rpcURL, "evm_revert", snapshotID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to revert snapshot: %w", err)
	}
	return nil
}
