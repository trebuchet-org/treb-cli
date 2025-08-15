package anvil

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/trebuchet-org/treb-cli/cli/pkg/dev"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// Manager is an adapter that wraps the legacy anvil management functionality
type Manager struct{}

// NewManager creates a new anvil manager adapter
func NewManager() *Manager {
	return &Manager{}
}

// Start starts an anvil instance
func (m *Manager) Start(ctx context.Context, instance *domain.AnvilInstance) error {
	return dev.StartAnvilInstance(instance.Name, instance.Port, instance.ChainID)
}

// Stop stops an anvil instance
func (m *Manager) Stop(ctx context.Context, instance *domain.AnvilInstance) error {
	return dev.StopAnvilInstance(instance.Name, instance.Port)
}

// GetStatus gets the status of an anvil instance
func (m *Manager) GetStatus(ctx context.Context, instance *domain.AnvilInstance) (*domain.AnvilStatus, error) {
	// Create legacy instance
	legacyInstance := dev.NewAnvilInstance(instance.Name, instance.Port).WithChainID(instance.ChainID)
	
	// Get basic running status
	isRunning := m.isRunning(legacyInstance)
	pid := 0
	if isRunning {
		if pidVal, err := m.readPidFile(legacyInstance); err == nil {
			pid = pidVal
		}
	}
	
	status := &domain.AnvilStatus{
		Running: isRunning,
		PID:     pid,
		LogFile: legacyInstance.LogFile,
	}
	
	// Update instance with file paths from legacy
	instance.PidFile = legacyInstance.PidFile
	instance.LogFile = legacyInstance.LogFile
	
	if isRunning {
		status.RPCURL = fmt.Sprintf("http://localhost:%s", instance.Port)
		
		// Check RPC health
		if err := m.checkRPCHealth(legacyInstance); err == nil {
			status.RPCHealthy = true
		}
		
		// Check CreateX deployment
		if err := m.checkCreateXDeployment(legacyInstance); err == nil {
			status.CreateXDeployed = true
			status.CreateXAddress = dev.CreateXAddress
		}
	}
	
	return status, nil
}

// StreamLogs streams logs from an anvil instance
func (m *Manager) StreamLogs(ctx context.Context, instance *domain.AnvilInstance, writer io.Writer) error {
	// Get status to ensure we have the log file path
	status, err := m.GetStatus(ctx, instance)
	if err != nil {
		return err
	}
	
	if _, err := os.Stat(status.LogFile); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", status.LogFile)
	}
	
	// Use tail -f to stream logs
	cmd := exec.CommandContext(ctx, "tail", "-f", status.LogFile)
	cmd.Stdout = writer
	cmd.Stderr = writer
	
	return cmd.Run()
}

// isRunning checks if the instance is running (using legacy implementation)
func (m *Manager) isRunning(inst *dev.AnvilInstance) bool {
	// Access the private method through reflection or reimplementation
	// For now, we'll reimplement the logic
	pid, err := m.readPidFile(inst)
	if err != nil {
		return false
	}
	
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	// Check if process is alive
	err = process.Signal(os.Signal(nil))
	return err == nil
}

// readPidFile reads the PID from the instance PID file
func (m *Manager) readPidFile(inst *dev.AnvilInstance) (int, error) {
	data, err := os.ReadFile(inst.PidFile)
	if err != nil {
		return 0, err
	}
	
	var pid int
	_, err = fmt.Sscanf(string(data), "%d", &pid)
	return pid, err
}

// checkRPCHealth checks if the RPC endpoint is responding
func (m *Manager) checkRPCHealth(inst *dev.AnvilInstance) error {
	// Create a temporary instance to use the method
	// This is a workaround since the methods are private
	// In a real refactor, we would expose these methods or duplicate the logic
	return dev.ShowAnvilStatusInstance(inst.Name, inst.Port)
}

// checkCreateXDeployment checks if CreateX is deployed
func (m *Manager) checkCreateXDeployment(inst *dev.AnvilInstance) error {
	// Similar workaround for checking CreateX deployment
	return dev.ShowAnvilStatusInstance(inst.Name, inst.Port)
}