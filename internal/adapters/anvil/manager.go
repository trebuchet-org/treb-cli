package anvil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

const (
	// Constants for anvil management
	DefaultAnvilPort = "8545"
	CreateXAddress   = "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed"
)

// Manager manages anvil instances without pkg dependencies
type Manager struct{}

// NewManager creates a new internal anvil manager
func NewManager() *Manager {
	return &Manager{}
}

// Start starts an anvil instance
func (m *Manager) Start(ctx context.Context, instance *domain.AnvilInstance) error {
	// Set defaults
	if instance.Name == "" {
		instance.Name = "anvil"
	}
	if instance.Port == "" {
		instance.Port = DefaultAnvilPort
	}

	// Set file paths
	m.setFilePaths(instance)

	// Check if already running
	if m.isRunning(instance) {
		return fmt.Errorf("anvil '%s' is already running (PID file exists at %s)", instance.Name, instance.PidFile)
	}

	// Build anvil command
	args := buildAnvilArgs(instance)
	cmd := exec.Command("anvil", args...)

	// Create log file
	logFile, err := os.Create(instance.LogFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start anvil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start anvil: %w", err)
	}

	// Write PID file
	if err := m.writePidFile(instance, cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Wait for anvil to be ready (forked anvils can take longer to start)
	maxWait := 5 * time.Second
	if instance.ForkURL != "" {
		maxWait = 30 * time.Second
	}
	if err := m.waitForReady(instance, maxWait); err != nil {
		_ = cmd.Process.Kill()
		_ = os.Remove(instance.PidFile)
		return fmt.Errorf("anvil failed to become ready: %w", err)
	}

	// Deploy CreateX
	if err := m.deployCreateX(instance); err != nil {
		// Don't fail, just warn
		fmt.Printf("Warning: Failed to deploy CreateX: %v\n", err)
	}

	return nil
}

// waitForReady polls the anvil RPC endpoint until it responds or the timeout is reached
func (m *Manager) waitForReady(instance *domain.AnvilInstance, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 200 * time.Millisecond
	for time.Now().Before(deadline) {
		if err := m.checkRPCHealth(instance); err == nil {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("anvil not responding after %s", timeout)
}

// Stop stops an anvil instance
func (m *Manager) Stop(ctx context.Context, instance *domain.AnvilInstance) error {
	// Set file paths
	m.setFilePaths(instance)

	if !m.isRunning(instance) {
		return nil
	}

	// Read PID
	pid, err := m.readPidFile(instance)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	// Find and kill process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Try graceful shutdown first
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Force kill if graceful shutdown fails
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Remove PID file
	if err := os.Remove(instance.PidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	return nil
}

// GetStatus gets the status of an anvil instance
func (m *Manager) GetStatus(ctx context.Context, instance *domain.AnvilInstance) (*domain.AnvilStatus, error) {
	// Set file paths
	m.setFilePaths(instance)

	status := &domain.AnvilStatus{
		LogFile: instance.LogFile,
	}

	// Check if running
	if m.isRunning(instance) {
		status.Running = true

		// Get PID
		if pid, err := m.readPidFile(instance); err == nil {
			status.PID = pid
		}

		// Set RPC URL
		status.RPCURL = fmt.Sprintf("http://localhost:%s", instance.Port)

		// Check RPC health
		if err := m.checkRPCHealth(instance); err == nil {
			status.RPCHealthy = true
		}

		// Check CreateX deployment
		if err := m.checkCreateXDeployment(instance); err == nil {
			status.CreateXDeployed = true
			status.CreateXAddress = CreateXAddress
		}
	}

	return status, nil
}

// StreamLogs streams logs from an anvil instance
func (m *Manager) StreamLogs(ctx context.Context, instance *domain.AnvilInstance, writer io.Writer) error {
	// Set file paths
	m.setFilePaths(instance)

	if _, err := os.Stat(instance.LogFile); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", instance.LogFile)
	}

	// Use tail -f to stream logs
	cmd := exec.CommandContext(ctx, "tail", "-f", instance.LogFile)
	cmd.Stdout = writer
	cmd.Stderr = writer

	return cmd.Run()
}

// buildAnvilArgs constructs the command-line arguments for starting an anvil instance
func buildAnvilArgs(instance *domain.AnvilInstance) []string {
	args := []string{"--port", instance.Port, "--host", "0.0.0.0"}
	if instance.ChainID != "" {
		args = append(args, "--chain-id", instance.ChainID)
	}
	if instance.ForkURL != "" {
		args = append(args, "--fork-url", instance.ForkURL)
	}
	return args
}

// TakeSnapshot takes an EVM snapshot on the given anvil instance and returns the snapshot ID
func (m *Manager) TakeSnapshot(ctx context.Context, instance *domain.AnvilInstance) (string, error) {
	m.setFilePaths(instance)

	req := rpcRequest{
		Jsonrpc: "2.0",
		Method:  "evm_snapshot",
		Params:  []interface{}{},
		ID:      1,
	}

	var resp rpcResponse
	if err := m.makeRPCCallWithResponse(instance, req, &resp); err != nil {
		return "", fmt.Errorf("evm_snapshot RPC call failed: %w", err)
	}

	snapshotID, ok := resp.Result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected evm_snapshot response type: %T", resp.Result)
	}

	return snapshotID, nil
}

// RevertSnapshot reverts the anvil instance to a previously taken EVM snapshot
func (m *Manager) RevertSnapshot(ctx context.Context, instance *domain.AnvilInstance, snapshotID string) error {
	m.setFilePaths(instance)

	req := rpcRequest{
		Jsonrpc: "2.0",
		Method:  "evm_revert",
		Params:  []interface{}{snapshotID},
		ID:      1,
	}

	var resp rpcResponse
	if err := m.makeRPCCallWithResponse(instance, req, &resp); err != nil {
		return fmt.Errorf("evm_revert RPC call failed: %w", err)
	}

	success, ok := resp.Result.(bool)
	if !ok {
		return fmt.Errorf("unexpected evm_revert response type: %T", resp.Result)
	}
	if !success {
		return fmt.Errorf("evm_revert returned false for snapshot %s", snapshotID)
	}

	return nil
}

// setFilePaths sets the PID and log file paths for an instance
func (m *Manager) setFilePaths(instance *domain.AnvilInstance) {
	if instance.Name == "" {
		instance.Name = "anvil"
	}
	if instance.Port == "" {
		instance.Port = DefaultAnvilPort
	}

	// If paths are already set (e.g. by the caller for fork instances), skip
	if instance.PidFile != "" && instance.LogFile != "" {
		return
	}

	// Use backward-compatible paths for default instance
	if instance.Name == "anvil" && instance.Port == DefaultAnvilPort {
		instance.PidFile = "/tmp/treb-anvil-pid"
		instance.LogFile = "/tmp/treb-anvil.log"
	} else {
		// Use per-instance files (fork instances with name "fork-<network>" get /tmp/treb-fork-<network>.pid)
		base := "/tmp"
		instance.PidFile = filepath.Join(base, fmt.Sprintf("treb-%s.pid", instance.Name))
		instance.LogFile = filepath.Join(base, fmt.Sprintf("treb-%s.log", instance.Name))
	}
}

// isRunning checks if the instance is running
func (m *Manager) isRunning(instance *domain.AnvilInstance) bool {
	pid, err := m.readPidFile(instance)
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Check if process is alive (signal 0 doesn't actually send a signal)
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// readPidFile reads the PID from the instance PID file
func (m *Manager) readPidFile(instance *domain.AnvilInstance) (int, error) {
	data, err := os.ReadFile(instance.PidFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %s", string(data))
	}

	return pid, nil
}

// writePidFile writes the PID to the instance PID file
func (m *Manager) writePidFile(instance *domain.AnvilInstance, pid int) error {
	return os.WriteFile(instance.PidFile, []byte(strconv.Itoa(pid)), 0644)
}

// deployCreateX deploys the CreateX factory
func (m *Manager) deployCreateX(instance *domain.AnvilInstance) error {
	// Fetch CreateX bytecode from mainnet
	bytecode, err := m.fetchCreateXBytecode()
	if err != nil {
		return fmt.Errorf("failed to fetch CreateX bytecode: %w", err)
	}

	// Use anvil_setCode to deploy CreateX at the known address
	req := rpcRequest{
		Jsonrpc: "2.0",
		Method:  "anvil_setCode",
		Params:  []interface{}{CreateXAddress, bytecode},
		ID:      1,
	}

	return m.makeRPCCall(instance, req)
}

// getCreateXCachePath returns the path to the cached CreateX bytecode
func getCreateXCachePath() string {
	return filepath.Join("/tmp", fmt.Sprintf("treb-createx-%s.bytecode", CreateXAddress))
}

// fetchCreateXBytecode fetches the CreateX bytecode from cache or mainnet
func (m *Manager) fetchCreateXBytecode() (string, error) {
	cachePath := getCreateXCachePath()

	// Try to read from cache first
	if cachedBytecode, err := os.ReadFile(cachePath); err == nil {
		// Validate cached bytecode
		bytecode := string(cachedBytecode)
		if bytecode != "" && bytecode != "0x" && len(bytecode) > 100 {
			return bytecode, nil
		}
	}

	// Cache miss or invalid, fetch from mainnet
	// Use public mainnet RPC to fetch CreateX bytecode
	mainnetRPC := "https://eth.llamarpc.com"

	req := rpcRequest{
		Jsonrpc: "2.0",
		Method:  "eth_getCode",
		Params:  []interface{}{CreateXAddress, "latest"},
		ID:      1,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpResp, err := http.Post(mainnetRPC, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", httpResp.StatusCode)
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", err
	}

	var resp rpcResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}

	if resp.Error != nil {
		return "", fmt.Errorf("RPC error: %s", resp.Error.Message)
	}

	bytecode, ok := resp.Result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected response type")
	}

	if bytecode == "0x" || bytecode == "" {
		return "", fmt.Errorf("CreateX contract not found at %s on mainnet", CreateXAddress)
	}

	// Cache the bytecode for future use
	if err := os.WriteFile(cachePath, []byte(bytecode), 0644); err != nil {
		// Log warning but don't fail - we have the bytecode
		fmt.Printf("Warning: Failed to cache CreateX bytecode: %v\n", err)
	}

	return bytecode, nil
}

// checkRPCHealth checks if the RPC endpoint is responding
func (m *Manager) checkRPCHealth(instance *domain.AnvilInstance) error {
	req := rpcRequest{
		Jsonrpc: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		ID:      1,
	}

	return m.makeRPCCall(instance, req)
}

// checkCreateXDeployment checks if CreateX is deployed
func (m *Manager) checkCreateXDeployment(instance *domain.AnvilInstance) error {
	req := rpcRequest{
		Jsonrpc: "2.0",
		Method:  "eth_getCode",
		Params:  []interface{}{CreateXAddress, "latest"},
		ID:      1,
	}

	var resp rpcResponse
	if err := m.makeRPCCallWithResponse(instance, req, &resp); err != nil {
		return err
	}

	code, ok := resp.Result.(string)
	if !ok {
		return fmt.Errorf("unexpected response type")
	}

	if code == "0x" || code == "" {
		return fmt.Errorf("no bytecode at address")
	}

	return nil
}

// makeRPCCall makes an RPC call without caring about the response
func (m *Manager) makeRPCCall(instance *domain.AnvilInstance, req rpcRequest) error {
	var resp rpcResponse
	return m.makeRPCCallWithResponse(instance, req, &resp)
}

// makeRPCCallWithResponse makes an RPC call and parses the response
func (m *Manager) makeRPCCallWithResponse(instance *domain.AnvilInstance, req rpcRequest, resp *rpcResponse) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpResp, err := http.Post(fmt.Sprintf("http://localhost:%s", instance.Port), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", httpResp.StatusCode)
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, resp); err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("RPC error: %s", resp.Error.Message)
	}

	return nil
}

// RPC types
type rpcRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type rpcResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
	ID      int         `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
