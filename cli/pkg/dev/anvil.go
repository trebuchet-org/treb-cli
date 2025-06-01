package dev

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
)

const (
	AnvilPidFile   = "/tmp/treb-anvil-pid"
	AnvilLogFile   = "/tmp/treb-anvil.log"
	AnvilPort      = "8545"
	CreateXAddress = "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed"
)

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

// StartAnvil starts a local anvil node with CreateX deployed
func StartAnvil() error {
	// Check if already running
	if isAnvilRunning() {
		return fmt.Errorf("anvil is already running (PID file exists at %s)", AnvilPidFile)
	}

	color.New(color.FgCyan, color.Bold).Println("🔨 Starting local anvil node...")

	// Start anvil process
	cmd := exec.Command("anvil", "--port", AnvilPort, "--host", "0.0.0.0")

	// Create log file
	logFile, err := os.Create(AnvilLogFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Redirect stdout and stderr to log file
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start anvil: %w", err)
	}

	// Write PID file
	if err := writePidFile(cmd.Process.Pid); err != nil {
		// Kill the process if we can't write PID file
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	color.New(color.FgGreen).Printf("✅ Anvil started with PID %d\n", cmd.Process.Pid)
	color.New(color.FgYellow).Printf("📋 Logs: %s\n", AnvilLogFile)
	color.New(color.FgBlue).Printf("🌐 RPC URL: http://localhost:%s\n", AnvilPort)

	// Wait a moment for anvil to start
	time.Sleep(2 * time.Second)

	// Deploy CreateX factory
	if err := deployCreateX(); err != nil {
		color.New(color.FgRed).Printf("⚠️  Warning: Failed to deploy CreateX: %v\n", err)
		color.New(color.FgYellow).Println("Deployments may fail without CreateX factory")
	} else {
		color.New(color.FgGreen).Printf("✅ CreateX factory deployed at %s\n", CreateXAddress)
	}

	return nil
}

// StopAnvil stops the anvil node
func StopAnvil() error {
	if !isAnvilRunning() {
		color.New(color.FgYellow).Println("Anvil is not running")
		return nil
	}

	pid, err := readPidFile()
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	color.New(color.FgCyan, color.Bold).Printf("🛑 Stopping anvil (PID %d)...\n", pid)

	// Kill the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, try SIGKILL
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Remove PID file
	if err := os.Remove(AnvilPidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	color.New(color.FgGreen).Println("✅ Anvil stopped")
	return nil
}

// RestartAnvil restarts the anvil node
func RestartAnvil() error {
	color.New(color.FgCyan, color.Bold).Println("🔄 Restarting anvil...")

	// Stop if running
	if isAnvilRunning() {
		if err := StopAnvil(); err != nil {
			return fmt.Errorf("failed to stop anvil: %w", err)
		}
		// Wait a moment for cleanup
		time.Sleep(1 * time.Second)
	}

	// Start
	return StartAnvil()
}

// ShowAnvilLogs shows the anvil logs
func ShowAnvilLogs() error {
	if !isAnvilRunning() {
		color.New(color.FgYellow).Println("Anvil is not running")
	}

	// Check if log file exists
	if _, err := os.Stat(AnvilLogFile); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", AnvilLogFile)
	}

	// Use tail command to show logs
	cmd := exec.Command("tail", "-f", AnvilLogFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	color.New(color.FgCyan, color.Bold).Printf("📋 Showing anvil logs (Ctrl+C to exit):\n")
	color.New(color.FgHiBlack).Printf("Log file: %s\n\n", AnvilLogFile)

	return cmd.Run()
}

// ShowAnvilStatus shows the status of the anvil node
func ShowAnvilStatus() error {
	running := isAnvilRunning()

	color.New(color.FgCyan, color.Bold).Println("📊 Anvil Status:")

	if running {
		pid, _ := readPidFile()
		color.New(color.FgGreen).Printf("Status: 🟢 Running (PID %d)\n", pid)
		color.New(color.FgBlue).Printf("RPC URL: http://localhost:%s\n", AnvilPort)
		color.New(color.FgYellow).Printf("Log file: %s\n", AnvilLogFile)

		// Check if RPC is responding
		if err := checkRPCHealth(); err != nil {
			color.New(color.FgRed).Printf("RPC Health: ❌ Not responding (%v)\n", err)
		} else {
			color.New(color.FgGreen).Println("RPC Health: ✅ Responding")
		}

		// Check CreateX deployment
		if err := checkCreateXDeployment(); err != nil {
			color.New(color.FgRed).Printf("CreateX Status: ❌ Not deployed (%v)\n", err)
		} else {
			color.New(color.FgGreen).Printf("CreateX Status: ✅ Deployed at %s\n", CreateXAddress)
		}
	} else {
		color.New(color.FgRed).Println("Status: 🔴 Not running")
		color.New(color.FgHiBlack).Printf("PID file: %s\n", AnvilPidFile)
		color.New(color.FgHiBlack).Printf("Log file: %s\n", AnvilLogFile)
	}

	return nil
}

// isAnvilRunning checks if anvil is running by checking the PID file
func isAnvilRunning() bool {
	pid, err := readPidFile()
	if err != nil {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Try to send signal 0 to check if process is alive
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// readPidFile reads the PID from the PID file
func readPidFile() (int, error) {
	data, err := os.ReadFile(AnvilPidFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %s", string(data))
	}

	return pid, nil
}

// writePidFile writes the PID to the PID file
func writePidFile(pid int) error {
	return os.WriteFile(AnvilPidFile, []byte(strconv.Itoa(pid)), 0644)
}

// fetchCreateXBytecode fetches the CreateX bytecode from mainnet
func fetchCreateXBytecode() (string, error) {
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

	return bytecode, nil
}

// deployCreateX deploys the CreateX factory using RPC
func deployCreateX() error {
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

	if err := makeRPCCall(req); err != nil {
		return fmt.Errorf("failed to deploy CreateX: %w", err)
	}

	return nil
}

// checkRPCHealth checks if the RPC endpoint is responding
func checkRPCHealth() error {
	req := RPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		ID:      1,
	}

	return makeRPCCall(req)
}

// checkCreateXDeployment checks if CreateX is deployed
func checkCreateXDeployment() error {
	req := RPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_getCode",
		Params:  []interface{}{CreateXAddress, "latest"},
		ID:      1,
	}

	var resp RPCResponse
	if err := makeRPCCallWithResponse(req, &resp); err != nil {
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
func makeRPCCall(req RPCRequest) error {
	var resp RPCResponse
	return makeRPCCallWithResponse(req, &resp)
}

// makeRPCCallWithResponse makes an RPC call and parses the response
func makeRPCCallWithResponse(req RPCRequest, resp *RPCResponse) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpResp, err := http.Post(fmt.Sprintf("http://localhost:%s", AnvilPort), "application/json", bytes.NewBuffer(jsonData))
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
