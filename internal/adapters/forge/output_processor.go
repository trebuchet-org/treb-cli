package forge

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
)

type Stage string

const (
	StageSimulating   Stage = "Simulating"
	StageBroadcasting Stage = "Broadcasting"
	StageCompleted    Stage = "Completed"
)

// ParsedEntity represents different types of parsed output
type ParsedEntity struct {
	Type    string
	Data    interface{}
	RawLine string
	Stage   Stage
}

// StageInfo tracks information about each stage
type StageInfo struct {
	Stage     Stage
	StartTime time.Time
	EndTime   time.Time
	Completed bool
	Skipped   bool
	Lines     int
}

// OutputProcessor handles real-time output processing
type OutputProcessor struct {
	debugDir     string
	ignoredCount int
	currentStage Stage
	spinner      *spinner.Spinner
	stages       []StageInfo
	hasReceipts  bool     // Track if we've seen any Receipt entities
	textOutput   []string // Collect non-JSON text output
	mu           sync.Mutex
}

// NewOutputProcessor creates a new output processor
func NewOutputProcessor(debugDir string) *OutputProcessor {
	// Create custom spinner with colors
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + string(StageSimulating)
	s.HideCursor = true

	return &OutputProcessor{
		debugDir:     debugDir,
		ignoredCount: 0,
		currentStage: "", // Start with empty stage so enterStage works properly
		spinner:      s,
		stages:       []StageInfo{},
		hasReceipts:  false,
		textOutput:   []string{},
	}
}

// ProcessOutput processes output in real-time, returning parsed entities via channel
func (op *OutputProcessor) ProcessOutput(reader io.Reader, entityChan chan<- ParsedEntity) error {
	scanner := bufio.NewScanner(reader)

	// Start with larger buffer for long lines
	const maxTokenSize = 200 * 1024 * 1024 // 200MB
	buf := make([]byte, maxTokenSize)
	scanner.Buffer(buf, maxTokenSize)

	// Start spinner
	op.spinner.Start()
	defer op.spinner.Stop()

	// Track current stage
	op.enterStage(StageSimulating)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Process the line
		if entity, parsed := op.parseLine(line); parsed {
			// Update stage based on parsed entity
			op.updateStageFromEntity(entity)

			if entity.Type == "UnknownJSON" {
				// Save JSON lines individually for debugging
				op.saveIgnoredLine(line)
			}

			// Send parsed entity
			entityChan <- entity
		} else if strings.TrimSpace(line) != "" {
			// Collect non-JSON text output
			op.mu.Lock()
			op.textOutput = append(op.textOutput, line)
			op.mu.Unlock()
		}

		// Update spinner
		op.updateSpinner()
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	// Complete current stage if not already completed
	op.completeCurrentStage()

	// Handle final stage cleanup
	op.mu.Lock()

	// Check if we have a Broadcasting stage
	hasBroadcastingStage := false
	for i := range op.stages {
		if op.stages[i].Stage == StageBroadcasting {
			hasBroadcastingStage = true
			// If broadcasting stage exists but no receipts, mark it as skipped
			if !op.hasReceipts {
				op.stages[i].Skipped = true
				if op.stages[i].EndTime.IsZero() {
					op.stages[i].EndTime = time.Now()
				}
			}
		}
	}

	// If we never entered broadcasting stage (dry-run without gas estimate), add it as skipped
	if !hasBroadcastingStage && op.currentStage == StageSimulating {
		op.stages = append(op.stages, StageInfo{
			Stage:     StageBroadcasting,
			StartTime: time.Now(),
			EndTime:   time.Now(),
			Completed: false,
			Skipped:   true,
		})
	}

	op.mu.Unlock()

	// Send collected text output as final entity
	op.mu.Lock()
	if len(op.textOutput) > 0 {
		entityChan <- ParsedEntity{
			Type:    "TextOutput",
			Data:    strings.Join(op.textOutput, "\n"),
			Stage:   op.currentStage,
			RawLine: "", // No single raw line for combined output
		}
	}
	op.mu.Unlock()

	op.printSummary()

	return nil
}

// parseLine attempts to parse a line into a known entity
func (op *OutputProcessor) parseLine(line string) (ParsedEntity, bool) {
	reader := strings.NewReader(line)
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields() // This enforces strict matching

	// Check if it's JSON
	if strings.HasPrefix(strings.TrimSpace(line), "{") {
		// Try to parse as ScriptOutput
		var scriptOutput forge.ScriptOutput
		if err := decoder.Decode(&scriptOutput); err == nil {
			if scriptOutput.RawLogs != nil {
				return ParsedEntity{
					Type:    "ScriptOutput",
					Data:    scriptOutput,
					Stage:   op.currentStage,
					RawLine: line,
				}, true
			}
		} else {
			reader.Reset(line)
		}

		// Try to parse as GasEstimate
		var gasEstimate forge.GasEstimate
		if err := decoder.Decode(&gasEstimate); err == nil {
			if gasEstimate.Chain != 0 {
				return ParsedEntity{
					Type:    "GasEstimate",
					Data:    gasEstimate,
					Stage:   op.currentStage,
					RawLine: line,
				}, true
			}
		} else {
			reader.Reset(line)
		}

		// Try to parse as Receipt
		var receipt forge.Receipt
		if err := decoder.Decode(&receipt); err == nil {
			return ParsedEntity{
				Type:    "Receipt",
				Data:    receipt,
				Stage:   op.currentStage,
				RawLine: line,
			}, true
		} else {
			reader.Reset(line)
		}

		// Try to parse as StatusOutput
		var statusOutput forge.StatusOutput
		if err := decoder.Decode(&statusOutput); err == nil {
			if statusOutput.Status != "" {
				return ParsedEntity{
					Type:    "StatusOutput",
					Data:    statusOutput,
					Stage:   op.currentStage,
					RawLine: line,
				}, true
			}
		} else {
			reader.Reset(line)
		}

		// Try to parse as TraceOutput
		var traceOutput forge.TraceOutput
		if err := decoder.Decode(&traceOutput); err == nil {
			if len(traceOutput.Arena) > 0 {
				return ParsedEntity{
					Type:    "TraceOutput",
					Data:    traceOutput,
					Stage:   op.currentStage,
					RawLine: line,
				}, true
			}
		} else {
			reader.Reset(line)
		}

		// It's JSON but we don't recognize it
		return ParsedEntity{
			Type:    "UnknownJSON",
			Data:    line,
			Stage:   op.currentStage,
			RawLine: line,
		}, true
	}

	return ParsedEntity{}, false
}

// updateStageFromEntity updates stage based on parsed entity type
func (op *OutputProcessor) updateStageFromEntity(entity ParsedEntity) {
	switch entity.Type {
	case "Receipt":
		// Track that we've seen receipts
		op.mu.Lock()
		op.hasReceipts = true
		op.mu.Unlock()
	case "GasEstimate":
		// After gas estimate, we move to broadcasting
		if op.currentStage == StageSimulating {
			op.enterStage(StageBroadcasting)
		}
	case "StatusOutput":
		// After status output, we're completed
		// Note: StatusOutput might come even in dry-run mode
		if op.currentStage == StageBroadcasting || op.currentStage == StageSimulating {
			op.enterStage(StageCompleted)
		}
	}
}

// enterStage transitions to a new stage
func (op *OutputProcessor) enterStage(stage Stage) {
	op.mu.Lock()
	defer op.mu.Unlock()

	// Don't re-enter the same stage
	if op.currentStage == stage {
		return
	}

	// Complete current stage if there is one
	if len(op.stages) > 0 && op.currentStage != "" {
		op.stages[len(op.stages)-1].EndTime = time.Now()
		op.stages[len(op.stages)-1].Completed = true
	}

	// Start new stage
	op.currentStage = stage
	op.stages = append(op.stages, StageInfo{
		Stage:     stage,
		StartTime: time.Now(),
		Completed: false,
		Skipped:   false,
	})

	go op.updateSpinner()
}

// completeCurrentStage marks the current stage as completed
func (op *OutputProcessor) completeCurrentStage() {
	op.mu.Lock()
	defer op.mu.Unlock()

	if len(op.stages) > 0 {
		lastStage := &op.stages[len(op.stages)-1]
		if !lastStage.Completed {
			lastStage.EndTime = time.Now()
			lastStage.Completed = true
		}
	}
}

// updateSpinner updates the spinner display
func (op *OutputProcessor) updateSpinner() {
	op.mu.Lock()
	defer op.mu.Unlock()

	// Build multi-line display
	var lines []string

	for _, stage := range op.stages {
		var icon string
		var stageColor *color.Color

		if stage.Skipped {
			icon = "⊘"
			stageColor = color.New(color.FgWhite, color.Faint)
		} else if stage.Completed {
			icon = "✓"
			stageColor = color.New(color.FgGreen)
		} else if stage.Stage == op.currentStage {
			icon = "●"
			stageColor = color.New(color.FgYellow)
		} else {
			icon = "○"
			stageColor = color.New(color.FgWhite)
		}

		duration := ""
		if !stage.EndTime.IsZero() {
			duration = fmt.Sprintf(" (%s)", stage.EndTime.Sub(stage.StartTime).Round(time.Millisecond))
		} else if stage.Stage == op.currentStage && !stage.Skipped {
			duration = fmt.Sprintf(" (%s)", time.Since(stage.StartTime).Round(time.Second))
		}

		lines = append(lines, fmt.Sprintf("%s %s%s", icon, stageColor.Sprint(stage.Stage), duration))
	}

	// Update spinner suffix with current stage info
	op.spinner.Suffix = fmt.Sprintf(" %s", strings.Join(lines, " → "))
}

// saveIgnoredLine saves a line that couldn't be parsed
func (op *OutputProcessor) saveIgnoredLine(line string) {
	op.mu.Lock()
	op.ignoredCount++
	count := op.ignoredCount
	op.mu.Unlock()

	filename := filepath.Join(op.debugDir, fmt.Sprintf("ignored-line%d.txt", count))
	if err := os.WriteFile(filename, []byte(line), 0644); err != nil {
		// Silently ignore write errors
		return
	}
}

// printSummary prints a summary of the execution
func (op *OutputProcessor) printSummary() {
	op.spinner.Stop()

	// Only show warning if there were parsing issues
	if op.ignoredCount > 0 {
		// Ignored lines have been saved to debug directory
	}
}
