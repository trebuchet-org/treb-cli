package forge

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/creack/pty"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ForgeAdapter handles Forge command execution with streaming output
type ForgeAdapter struct {
	log         *slog.Logger
	projectRoot string
}

// NewForgeAdapter creates a new forge executor
func NewForgeAdapter(projectRoot string, log *slog.Logger) *ForgeAdapter {
	return &ForgeAdapter{
		log:         log.With("component", "ForgeAdapter"),
		projectRoot: projectRoot,
	}
}

// Build runs forge build with proper output handling
func (f *ForgeAdapter) Build() error {
	start := time.Now()
	f.log.Debug("running forge build", "dir", f.projectRoot)

	cmd := exec.Command("forge", "build")
	cmd.Dir = f.projectRoot

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	if err != nil {
		f.log.Error("forge build failed", "error", err, "output", string(output), "duration", duration)
		// Only print error details if build actually failed
		return fmt.Errorf("forge build failed: %w\nOutput: %s", err, string(output))
	}

	f.log.Debug("forge build completed successfully", "duration", duration)
	// Don't print anything on success - let the caller handle UI
	return nil
}

// Run executes a Foundry script with the given options
func (f *ForgeAdapter) RunScript(ctx context.Context, config usecase.RunScriptConfig) (*forge.RunResult, error) {
	// Build forge command arguments
	args := f.buildArgs(config)

	// Build environment variables
	env := f.buildEnv(config)

	f.log.Debug("Running forge script", "args", args, "env", env)

	// Execute the script
	cmd := exec.CommandContext(ctx, "forge", args...)
	cmd.Dir = f.projectRoot
	cmd.Env = append(os.Environ(), env...)

	// Start with PTY for proper color handling
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}
	defer func() {
		// Close PTY after command finishes to avoid read errors
		_ = ptyFile.Close()
	}()

	result := &forge.RunResult{
		DryRun:    config.DryRun,
		Script:    config.Script,
		Success:   true, // Will be updated based on command exit
		Namespace: config.Namespace,
		Network:   config.Network.Name,
		ChainID:   config.Network.ChainID,
		Senders:   config.SenderScriptConfig,
	}

	// Debug mode: direct copy to stdout
	if config.Debug && !config.DebugJSON {
		// Simple copy for debug mode
		_, _ = io.Copy(os.Stdout, ptyFile)

		// Wait for command to finish
		if err := cmd.Wait(); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("forge script failed: %w", err)
		}

		return result, nil
	}

	// Normal mode: process output with scanner
	// Create debug directory for this run
	runUUID := fmt.Sprintf("%d", time.Now().Unix())
	debugDir := filepath.Join(f.projectRoot, "out", ".treb-debug", runUUID)
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		fmt.Printf("Warning: failed to create debug directory: %v\n", err)
		// Continue anyway
		debugDir = ""
	}

	// Create channels for parsed entities
	entityChan := make(chan ParsedEntity, 100)
	errChan := make(chan error, 1)

	// Collect all output for raw output
	var outputBuffer bytes.Buffer
	teeReader := io.TeeReader(ptyFile, &outputBuffer)

	// Start output processor
	processor := NewOutputProcessor(debugDir)

	// Process output in goroutine
	processingDone := make(chan struct{})
	go func() {
		defer func() {
			close(entityChan)
			close(errChan)
			close(processingDone)
		}()

		if err := processor.ProcessOutput(teeReader, entityChan); err != nil {
			errChan <- err
		}
	}()

	// Collect parsed entities
	parsedOutput := &forge.ParsedOutput{
		ConsoleLogs:  []string{},
		TraceOutputs: []forge.TraceOutput{},
	}

	for entity := range entityChan {
		switch entity.Type {
		case "ScriptOutput":
			if output, ok := entity.Data.(forge.ScriptOutput); ok {
				parsedOutput.ScriptOutput = &output
				// Console logs are already in output.Logs
				parsedOutput.ConsoleLogs = append(parsedOutput.ConsoleLogs, output.Logs...)
			}
		case "GasEstimate":
			if gas, ok := entity.Data.(forge.GasEstimate); ok {
				parsedOutput.GasEstimate = &gas
			}
		case "StatusOutput":
			if status, ok := entity.Data.(forge.StatusOutput); ok {
				parsedOutput.StatusOutput = &status
				// Extract broadcast path
				if status.Transactions != "" {
					result.BroadcastPath = status.Transactions
				}
			}
		case "TraceOutput":
			if trace, ok := entity.Data.(forge.TraceOutput); ok {
				parsedOutput.TraceOutputs = append(parsedOutput.TraceOutputs, trace)
			}
		case "TextOutput":
			if text, ok := entity.Data.(string); ok {
				parsedOutput.TextOutput = text
			}
		case "Receipt":
			if receipt, ok := entity.Data.(forge.Receipt); ok {
				if parsedOutput.Receipts == nil {
					parsedOutput.Receipts = []forge.Receipt{}
				}
				parsedOutput.Receipts = append(parsedOutput.Receipts, receipt)
			}
		}
	}

	// Wait for command to finish first
	cmdErr := cmd.Wait()

	// Wait for processing to complete
	<-processingDone

	// Check for processing errors
	select {
	case err := <-errChan:
		if err != nil {
			result.Error = fmt.Errorf("output processing error: %w", err)
		}
	default:
	}

	// Handle command error after processing is done
	if cmdErr != nil {
		result.Success = false
		if result.Error == nil {
			result.Error = fmt.Errorf("forge script failed: %w", cmdErr)
		}
	}

	// Set results
	result.RawOutput = outputBuffer.Bytes()
	result.ParsedOutput = parsedOutput

	// Check for script failure in text output even if exit code is 0
	// This handles platform differences where forge might exit with 0 even on revert
	if parsedOutput.TextOutput != "" && result.Error == nil {
		lowerOutput := strings.ToLower(parsedOutput.TextOutput)
		if strings.Contains(lowerOutput, "error:") ||
			strings.Contains(lowerOutput, "revert") ||
			strings.Contains(lowerOutput, "script failed") {
			result.Success = false
			result.Error = fmt.Errorf("script execution failed")
		}
	}

	// Print text output if script failed or in debug/verbose mode
	if parsedOutput.TextOutput != "" && (result.Error != nil || config.Debug) {
		fmt.Println("\n📝 Forge Output:")
		fmt.Println(strings.Repeat("─", 50))
		fmt.Println(parsedOutput.TextOutput)
		fmt.Println(strings.Repeat("─", 50))
	}

	// Save debug output if requested
	if config.Debug && config.DebugJSON && len(result.RawOutput) > 0 {
		f.saveDebugOutput(result.RawOutput)
	}

	return result, nil
}

// buildArgs builds the forge script command arguments
func (f *ForgeAdapter) buildArgs(config usecase.RunScriptConfig) []string {
	args := []string{"script", config.Script.Path, "--ffi"}

	// Network configuration
	args = append(args, "--rpc-url", config.Network.Name)

	// Broadcast/dry run
	if !config.DryRun {
		args = append(args, "--broadcast")
	}

	// Ledger flag if required
	if config.SenderScriptConfig.UseLedger {
		args = append(args, "--ledger")
	}

	if config.SenderScriptConfig.UseTrezor {
		args = append(args, "--trezor")
	}

	if len(config.SenderScriptConfig.DerivationPaths) > 0 {
		args = append(args, "--mnemonic-derivation-paths", strings.Join(config.SenderScriptConfig.DerivationPaths, ","))
	}

	// Libraries for linking
	for _, lib := range config.Libraries {
		args = append(args, "--libraries", lib)
	}

	// JSON output
	if config.DebugJSON || !config.Debug {
		args = append(args, "--json")
	}

	if config.Slow {
		args = append(args, "--slow")
	}
	args = append(args, "-vvvv")

	return args
}

// buildEnv builds environment variable array
func (f *ForgeAdapter) buildEnv(config usecase.RunScriptConfig) []string {
	env := make(map[string]string)
	maps.Copy(env, config.Parameters)

	// Profile
	env["FOUNDRY_PROFILE"] = config.Namespace
	env["NAMESPACE"] = config.Namespace
	env["NETWORK"] = config.Network.Name
	env["DRYRUN"] = strconv.FormatBool(config.DryRun || config.Debug || config.DebugJSON)
	env["SENDER_CONFIGS"] = config.SenderScriptConfig.EncodedConfig

	if config.Debug {
		env["QUIET"] = "true"
	}

	var envStrings []string
	for k, v := range env {
		envStrings = append(envStrings, fmt.Sprintf("%s=%s", k, v))

	}

	return envStrings
}

// saveDebugOutput saves raw output for debugging
func (f *ForgeAdapter) saveDebugOutput(output []byte) {
	runUUID := fmt.Sprintf("%d", time.Now().Unix())
	debugDir := filepath.Join(f.projectRoot, "out", ".treb-debug", runUUID)

	if err := os.MkdirAll(debugDir, 0755); err != nil {
		fmt.Printf("Warning: failed to create debug directory: %v\n", err)
		return
	}

	// Save raw output
	rawPath := filepath.Join(debugDir, "raw-output.json")
	if err := os.WriteFile(rawPath, output, 0644); err != nil {
		fmt.Printf("Warning: failed to write raw output: %v\n", err)
	} else {
		fmt.Printf("Debug output saved to: %s\n", debugDir)
	}
}
