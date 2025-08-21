package helpers

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"strings"

	"github.com/google/uuid"
	"github.com/pelletier/go-toml/v2"
	"github.com/trebuchet-org/treb-cli/cli/pkg/dev"
)

// TestContext represents an isolated test environment
type TestContext struct {
	ID          string
	WorkDir     string
	AnvilNodes  map[string]*AnvilNode
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

// ProgressReporter handles progress reporting with spinners
type ProgressReporter struct {
	mu      sync.Mutex
	current string
	done    bool
}

var (
	globalPool     *TestContextPool
	globalReporter *ProgressReporter
	// IsParallelMode indicates whether tests are running in parallel mode
	IsParallelMode bool
)

// GetGlobalPool returns the global test context pool
func GetGlobalPool() *TestContextPool {
	return globalPool
}

// InitializeTestPool creates a global test context pool
func InitializeTestPool(poolSize int) error {
	if globalPool != nil {
		return nil // Already initialized
	}

	// Set parallel mode flag
	IsParallelMode = true

	reporter := &ProgressReporter{}
	globalReporter = reporter

	// Start progress reporter
	go reporter.Run()

	reporter.SetStatus("Building binaries...")
	if err := buildBinaries(); err != nil {
		return fmt.Errorf("failed to build binaries: %w", err)
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
	reporter.SetStatus("Building contracts...")
	cmd := exec.Command("forge", "build")
	cmd.Dir = baseProject
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build contracts: %w", err)
	}

	// Create contexts
	for i := 0; i < poolSize; i++ {
		reporter.SetStatus(fmt.Sprintf("Preparing test context %d/%d...", i+1, poolSize))
		ctx, err := pool.createContext()
		if err != nil {
			// Clean up already created contexts
			for _, c := range pool.contexts {
				c.Cleanup()
			}
			return fmt.Errorf("failed to create context %d: %w", i, err)
		}
		pool.contexts = append(pool.contexts, ctx)
	}

	globalPool = pool
	reporter.SetStatus("Test environment ready")
	reporter.Stop()
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

	// Start first node
	if err := dev.StartAnvilInstance(node1Name, fmt.Sprintf("%d", port1), "31337"); err != nil {
		return nil, fmt.Errorf("failed to start anvil node 1: %w", err)
	}
	ctx.AnvilNodes["anvil-31337"] = &AnvilNode{
		AnvilInstance: dev.NewAnvilInstance(node1Name, fmt.Sprintf("%d", port1)).WithChainID("31337"),
		URL:           fmt.Sprintf("http://127.0.0.1:%d", port1),
	}

	// Start second node
	if err := dev.StartAnvilInstance(node2Name, fmt.Sprintf("%d", port2), "31338"); err != nil {
		dev.StopAnvilInstance(node1Name, fmt.Sprintf("%d", port1))
		return nil, fmt.Errorf("failed to start anvil node 2: %w", err)
	}
	ctx.AnvilNodes["anvil-31338"] = &AnvilNode{
		AnvilInstance: dev.NewAnvilInstance(node2Name, fmt.Sprintf("%d", port2)).WithChainID("31338"),
		URL:           fmt.Sprintf("http://127.0.0.1:%d", port2),
	}

	return ctx, nil
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

	for {
		for _, ctx := range p.contexts {
			if !ctx.inUse {
				ctx.inUse = true
				// Create TrebContext for the test
				version := GetBinaryVersionFromEnv()
				ctx.TrebContext = &TrebContext{
					t:             t,
					Network:       "anvil-31337",
					Namespace:     "default",
					BinaryVersion: version,
					workDir:       ctx.WorkDir,
				}
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
func (p *TestContextPool) Release(ctx *TestContext) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clean up the context
	ctx.Clean()
	ctx.inUse = false
	ctx.TrebContext = nil

	// Signal that a context is available
	p.cond.Signal()
}

// Clean cleans up a test context for reuse
func (ctx *TestContext) Clean() {
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
}

// Cleanup destroys the test context
func (ctx *TestContext) Cleanup() {
	// Stop anvil nodes
	for name, node := range ctx.AnvilNodes {
		dev.StopAnvilInstance(node.Name, node.Port)
		delete(ctx.AnvilNodes, name)
	}

	// Remove work directory
	os.RemoveAll(ctx.WorkDir)
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

// ProgressReporter implementation
func (r *ProgressReporter) SetStatus(status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.current = status
}

func (r *ProgressReporter) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.done = true
}

func (r *ProgressReporter) Run() {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	for {
		r.mu.Lock()
		if r.done {
			r.mu.Unlock()
			fmt.Printf("\r\033[K") // Clear line
			break
		}
		status := r.current
		r.mu.Unlock()

		if status != "" {
			fmt.Printf("\r%s %s", spinner[i], status)
			i = (i + 1) % len(spinner)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// ParallelIsolatedTest runs a test using the context pool
func ParallelIsolatedTest(t *testing.T, name string, fn func(t *testing.T, ctx *TrebContext)) {
	t.Run(name, func(t *testing.T) {
		t.Parallel() // Enable parallel execution

		if globalPool == nil {
			t.Fatal("Test pool not initialized. Call InitializeTestPool in TestMain")
		}

		// Acquire a context from the pool
		testCtx := globalPool.Acquire(t)
		defer globalPool.Release(testCtx)

		// Change to the test's work directory
		oldWd, _ := os.Getwd()
		os.Chdir(testCtx.WorkDir)
		defer os.Chdir(oldWd)

		// Run the test
		fn(t, testCtx.TrebContext)
	})
}
