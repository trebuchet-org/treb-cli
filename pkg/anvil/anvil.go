package anvil

import (
	"bytes"
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

	"github.com/fatih/color"
)

const (
	// Backward-compatible defaults
	AnvilPidFile   = "/tmp/treb-anvil-pid"
	AnvilLogFile   = "/tmp/treb-anvil.log"
	AnvilPort      = "8545"
	CreateXAddress = "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed"
)

// AnvilInstance represents a named local anvil instance
type AnvilInstance struct {
	Name    string
	Port    string
	ChainID string
	PidFile string
	LogFile string
}

// NewAnvilInstance creates a new instance descriptor with sensible defaults
func NewAnvilInstance(name, port string) *AnvilInstance {
	if strings.TrimSpace(name) == "" {
		name = "anvil"
	}
	if strings.TrimSpace(port) == "" {
		port = AnvilPort
	}
	// Keep old single-instance files for the default name to remain backward compatible
	var pidPath string
	var logPath string
	if name == "anvil" && port == AnvilPort {
		pidPath = AnvilPidFile
		logPath = AnvilLogFile
	} else {
		// Use per-instance files under /tmp with the instance name
		base := "/tmp"
		pidPath = filepath.Join(base, fmt.Sprintf("treb-%s.pid", name))
		logPath = filepath.Join(base, fmt.Sprintf("treb-%s.log", name))
	}
	return &AnvilInstance{
		Name:    name,
		Port:    port,
		PidFile: pidPath,
		LogFile: logPath,
	}
}

// WithChainID returns a copy of the instance with the provided chain ID set
func (a *AnvilInstance) WithChainID(chainID string) *AnvilInstance {
	a2 := *a
	a2.ChainID = strings.TrimSpace(chainID)
	return &a2
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      int         `json:"id"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// StartAnvil starts the default anvil instance (backward compatible)
func StartAnvil() error { return StartAnvilInstance("anvil", AnvilPort, "") }

// StartAnvilInstance starts a named anvil instance on the given port with CreateX deployed
func StartAnvilInstance(name, port, chainID string) error {
	inst := NewAnvilInstance(name, port).WithChainID(chainID)
	if inst.isRunning() {
		return fmt.Errorf("anvil '%s' is already running (PID file exists at %s)", inst.Name, inst.PidFile)
	}

	// color.New(color.FgCyan, color.Bold).Printf("🔨 Starting local anvil node '%s' on port %s...\n", inst.Name, inst.Port)

	args := []string{"--port", inst.Port, "--host", "0.0.0.0"}
	if inst.ChainID != "" {
		args = append(args, "--chain-id", inst.ChainID)
	}
	cmd := exec.Command("anvil", args...)

	logFile, err := os.Create(inst.LogFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start anvil: %w", err)
	}

	if err := inst.writePidFile(cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// color.New(color.FgGreen).Printf("✅ Anvil '%s' started with PID %d\n", inst.Name, cmd.Process.Pid)
	// color.New(color.FgYellow).Printf("📋 Logs: %s\n", inst.LogFile)
	// color.New(color.FgBlue).Printf("🌐 RPC URL: http://localhost:%s\n", inst.Port)

	time.Sleep(200 * time.Millisecond)

	if err := inst.deployCreateX(); err != nil {
		return fmt.Errorf("failed to deploy CreateX: %v", err)
	}

	return nil
}

// StopAnvil stops the default anvil instance
func StopAnvil() error { return StopAnvilInstance("anvil", AnvilPort) }

// StopAnvilInstance stops a named anvil instance
func StopAnvilInstance(name, port string) error {
	inst := NewAnvilInstance(name, port)
	if !inst.isRunning() {
		color.New(color.FgYellow).Printf("Anvil '%s' is not running\n", inst.Name)
		return nil
	}

	pid, err := inst.readPidFile()
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	// color.New(color.FgCyan, color.Bold).Printf("🛑 Stopping anvil '%s' (PID %d)...\n", inst.Name, pid)

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Wait for process to actually exit (with timeout)
	done := make(chan struct{})
	go func() {
		process.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		// Force kill if SIGTERM didn't work in time
		_ = process.Kill()
		<-done
	}

	if err := os.Remove(inst.PidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	// color.New(color.FgGreen).Println("✅ Anvil stopped")
	return nil
}

// RestartAnvil restarts the default anvil instance
func RestartAnvil() error { return RestartAnvilInstance("anvil", AnvilPort, "") }

// RestartAnvilInstance restarts a named anvil instance
func RestartAnvilInstance(name, port, chainID string) error {
	inst := NewAnvilInstance(name, port).WithChainID(chainID)
	color.New(color.FgCyan, color.Bold).Printf("🔄 Restarting anvil '%s'...\n", inst.Name)

	if inst.isRunning() {
		if err := StopAnvilInstance(inst.Name, inst.Port); err != nil {
			return fmt.Errorf("failed to stop anvil: %w", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	return StartAnvilInstance(inst.Name, inst.Port, inst.ChainID)
}

// ShowAnvilLogs shows the default anvil logs
func ShowAnvilLogs() error { return ShowAnvilLogsInstance("anvil", AnvilPort) }

// ShowAnvilLogsInstance shows logs for a named instance
func ShowAnvilLogsInstance(name, port string) error {
	inst := NewAnvilInstance(name, port)
	if !inst.isRunning() {
		color.New(color.FgYellow).Printf("Anvil '%s' is not running\n", inst.Name)
	}
	if _, err := os.Stat(inst.LogFile); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", inst.LogFile)
	}
	cmd := exec.Command("tail", "-f", inst.LogFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	color.New(color.FgCyan, color.Bold).Printf("📋 Showing anvil '%s' logs (Ctrl+C to exit):\n", inst.Name)
	color.New(color.FgHiBlack).Printf("Log file: %s\n\n", inst.LogFile)
	return cmd.Run()
}

// ShowAnvilStatus shows the status of the default anvil instance
func ShowAnvilStatus() error { return ShowAnvilStatusInstance("anvil", AnvilPort) }

// ShowAnvilStatusInstance shows the status of a named anvil instance
func ShowAnvilStatusInstance(name, port string) error {
	inst := NewAnvilInstance(name, port)
	running := inst.isRunning()

	color.New(color.FgCyan, color.Bold).Printf("📊 Anvil Status ('%s'):\n", inst.Name)

	if running {
		pid, _ := inst.readPidFile()
		color.New(color.FgGreen).Printf("Status: 🟢 Running (PID %d)\n", pid)
		color.New(color.FgBlue).Printf("RPC URL: http://localhost:%s\n", inst.Port)
		color.New(color.FgYellow).Printf("Log file: %s\n", inst.LogFile)

		if err := inst.checkRPCHealth(); err != nil {
			color.New(color.FgRed).Printf("RPC Health: ❌ Not responding (%v)\n", err)
		} else {
			color.New(color.FgGreen).Println("RPC Health: ✅ Responding")
		}

		if err := inst.checkCreateXDeployment(); err != nil {
			color.New(color.FgRed).Printf("CreateX Status: ❌ Not deployed (%v)\n", err)
		} else {
			color.New(color.FgGreen).Printf("CreateX Status: ✅ Deployed at %s\n", CreateXAddress)
		}
	} else {
		color.New(color.FgRed).Println("Status: 🔴 Not running")
		color.New(color.FgHiBlack).Printf("PID file: %s\n", inst.PidFile)
		color.New(color.FgHiBlack).Printf("Log file: %s\n", inst.LogFile)
	}

	return nil
}

// isRunning checks if this instance is running by checking its PID file
func (a *AnvilInstance) isRunning() bool {
	pid, err := a.readPidFile()
	if err != nil {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// readPidFile reads the PID from the instance PID file
func (a *AnvilInstance) readPidFile() (int, error) {
	data, err := os.ReadFile(a.PidFile)
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
func (a *AnvilInstance) writePidFile(pid int) error {
	return os.WriteFile(a.PidFile, []byte(strconv.Itoa(pid)), 0644)
}

// getCreateXCachePath returns the path to the cached CreateX bytecode
func getCreateXCachePath() string {
	return filepath.Join("/tmp", fmt.Sprintf("treb-createx-%s.bytecode", CreateXAddress))
}

// fetchCreateXBytecode fetches the CreateX bytecode from cache or mainnet
func fetchCreateXBytecode() (string, error) {
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

	req := RPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_getCode",
		Params:  []interface{}{CreateXAddress, "latest"},
		ID:      1,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpResp, err := http.Post(mainnetRPC, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", httpResp.StatusCode)
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var resp RPCResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.Error != nil {
		return "", fmt.Errorf("RPC error: %s", resp.Error.Message)
	}

	bytecode, ok := resp.Result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected response type: %T", resp.Result)
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

// deployCreateX deploys the CreateX factory using RPC
func (a *AnvilInstance) deployCreateX() error {
	// Fetch CreateX bytecode from mainnet
	bytecode, err := fetchCreateXBytecode()
	if err != nil {
		return fmt.Errorf("failed to fetch CreateX bytecode: %w", err)
	}

	// Use anvil_setCode to deploy CreateX at the known address
	params := []interface{}{
		CreateXAddress,
		bytecode,
	}

	req := RPCRequest{
		Jsonrpc: "2.0",
		Method:  "anvil_setCode",
		Params:  params,
		ID:      1,
	}

	if err := a.makeRPCCall(req); err != nil {
		return fmt.Errorf("failed to deploy CreateX: %w", err)
	}

	return nil
}

// checkRPCHealth checks if the RPC endpoint is responding
func (a *AnvilInstance) checkRPCHealth() error {
	req := RPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		ID:      1,
	}

	return a.makeRPCCall(req)
}

// checkCreateXDeployment checks if CreateX is deployed
func (a *AnvilInstance) checkCreateXDeployment() error {
	req := RPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_getCode",
		Params:  []interface{}{CreateXAddress, "latest"},
		ID:      1,
	}

	var resp RPCResponse
	if err := a.makeRPCCallWithResponse(req, &resp); err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("RPC error: %s", resp.Error.Message)
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
func (a *AnvilInstance) makeRPCCall(req RPCRequest) error {
	var resp RPCResponse
	return a.makeRPCCallWithResponse(req, &resp)
}

// makeRPCCallWithResponse makes an RPC call and parses the response
func (a *AnvilInstance) makeRPCCallWithResponse(req RPCRequest, resp *RPCResponse) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpResp, err := http.Post(fmt.Sprintf("http://localhost:%s", a.Port), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", httpResp.StatusCode)
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, resp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("RPC error: %s", resp.Error.Message)
	}

	return nil
}
