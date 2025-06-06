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
)

// Stage represents the current execution stage
type Stage string

const (
	StageInitializing Stage = "Initializing"
	StageCompiling    Stage = "Compiling"
	StageSimulating   Stage = "Simulating"
	StageBroadcasting Stage = "Broadcasting"
	StageCompleted    Stage = "Completed"
)

// ParsedEntity represents different types of parsed output
type ParsedEntity struct {
	Type   string
	Data   interface{}
	Stage  Stage
	RawLine string
}

// OutputProcessor handles real-time output processing
type OutputProcessor struct {
	debugDir     string
	ignoredCount int
	currentStage Stage
	spinner      *spinner.Spinner
	stages       []StageInfo
	mu           sync.Mutex
}

// StageInfo tracks information about each stage
type StageInfo struct {
	Stage     Stage
	StartTime time.Time
	EndTime   time.Time
	Completed bool
	Lines     int
}

// NewOutputProcessor creates a new output processor
func NewOutputProcessor(debugDir string) *OutputProcessor {
	// Create custom spinner with colors
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + string(StageInitializing)
	s.HideCursor = true
	
	return &OutputProcessor{
		debugDir:     debugDir,
		ignoredCount: 0,
		currentStage: StageInitializing,
		spinner:      s,
		stages:       []StageInfo{},
	}
}

// ProcessOutput processes output in real-time, returning parsed entities via channel
func (op *OutputProcessor) ProcessOutput(reader io.Reader, entityChan chan<- ParsedEntity) error {
	scanner := bufio.NewScanner(reader)
	
	// Start with larger buffer for long lines
	const maxTokenSize = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, maxTokenSize)
	scanner.Buffer(buf, maxTokenSize)
	
	// Start spinner
	op.spinner.Start()
	defer op.spinner.Stop()
	
	// Track current stage
	op.enterStage(StageInitializing)
	
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		// Try to detect stage changes from content
		op.detectStageFromContent(line)
		
		// Process the line
		if entity, parsed := op.parseLine(line); parsed {
			// Update stage based on parsed entity
			op.updateStageFromEntity(entity)
			
			// Send parsed entity
			select {
			case entityChan <- entity:
			default:
				// Channel might be full, log warning
				fmt.Printf("Warning: entity channel full, dropping entity\n")
			}
		} else if strings.TrimSpace(line) != "" {
			// Save unparsed lines
			op.saveIgnoredLine(line)
		}
		
		// Update spinner
		op.updateSpinner()
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}
	
	// Mark completion
	op.enterStage(StageCompleted)
	op.printSummary()
	
	return nil
}

// parseLine attempts to parse a line into a known entity
func (op *OutputProcessor) parseLine(line string) (ParsedEntity, bool) {
	// Remove ANSI color codes for parsing
	cleanLine := stripANSI(line)
	
	// Check if it's JSON
	if strings.HasPrefix(strings.TrimSpace(cleanLine), "{") {
		// Try to parse as ScriptOutput
		var scriptOutput ScriptOutput
		if err := json.Unmarshal([]byte(cleanLine), &scriptOutput); err == nil {
			if scriptOutput.RawLogs != nil {
				return ParsedEntity{
					Type:    "ScriptOutput",
					Data:    scriptOutput,
					Stage:   op.currentStage,
					RawLine: line,
				}, true
			}
		}
		
		// Try to parse as GasEstimate
		var gasEstimate GasEstimate
		if err := json.Unmarshal([]byte(cleanLine), &gasEstimate); err == nil {
			if gasEstimate.Chain != 0 {
				return ParsedEntity{
					Type:    "GasEstimate",
					Data:    gasEstimate,
					Stage:   op.currentStage,
					RawLine: line,
				}, true
			}
		}
		
		// Try to parse as StatusOutput
		var statusOutput StatusOutput
		if err := json.Unmarshal([]byte(cleanLine), &statusOutput); err == nil {
			if statusOutput.Status != "" {
				return ParsedEntity{
					Type:    "StatusOutput",
					Data:    statusOutput,
					Stage:   op.currentStage,
					RawLine: line,
				}, true
			}
		}
		
		// It's JSON but we don't recognize it
		return ParsedEntity{
			Type:    "UnknownJSON",
			Data:    cleanLine,
			Stage:   op.currentStage,
			RawLine: line,
		}, true
	}
	
	// Check for console.log
	if strings.Contains(line, "console.log") || strings.Contains(line, "Logs:") {
		return ParsedEntity{
			Type:    "ConsoleLog",
			Data:    line,
			Stage:   op.currentStage,
			RawLine: line,
		}, true
	}
	
	return ParsedEntity{}, false
}

// detectStageFromContent tries to detect stage changes from line content
func (op *OutputProcessor) detectStageFromContent(line string) {
	lower := strings.ToLower(line)
	
	if strings.Contains(lower, "compiling") {
		op.enterStage(StageCompiling)
	} else if strings.Contains(lower, "simulating") || strings.Contains(lower, "simulation") {
		op.enterStage(StageSimulating)
	} else if strings.Contains(lower, "broadcasting") || strings.Contains(lower, "broadcast") {
		op.enterStage(StageBroadcasting)
	}
}

// updateStageFromEntity updates stage based on parsed entity type
func (op *OutputProcessor) updateStageFromEntity(entity ParsedEntity) {
	switch entity.Type {
	case "ScriptOutput":
		// Script output usually means we're in simulation
		if op.currentStage == StageInitializing || op.currentStage == StageCompiling {
			op.enterStage(StageSimulating)
		}
	case "StatusOutput":
		// Status output often indicates broadcast completion
		if status, ok := entity.Data.(StatusOutput); ok {
			if status.Status == "success" && op.currentStage != StageBroadcasting {
				op.enterStage(StageBroadcasting)
			}
		}
	}
}

// enterStage transitions to a new stage
func (op *OutputProcessor) enterStage(stage Stage) {
	op.mu.Lock()
	defer op.mu.Unlock()
	
	if op.currentStage == stage {
		return
	}
	
	// Complete current stage
	if len(op.stages) > 0 {
		op.stages[len(op.stages)-1].EndTime = time.Now()
		op.stages[len(op.stages)-1].Completed = true
	}
	
	// Start new stage
	op.currentStage = stage
	op.stages = append(op.stages, StageInfo{
		Stage:     stage,
		StartTime: time.Now(),
		Completed: false,
	})
	
	// Update display
	op.updateSpinner()
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
		
		if stage.Completed {
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
		} else if stage.Stage == op.currentStage {
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
	
	fmt.Println("\n📊 Execution Summary:")
	fmt.Println(strings.Repeat("─", 50))
	
	totalDuration := time.Duration(0)
	for _, stage := range op.stages {
		var duration time.Duration
		if !stage.EndTime.IsZero() {
			duration = stage.EndTime.Sub(stage.StartTime)
		} else {
			duration = time.Since(stage.StartTime)
		}
		totalDuration += duration
		
		icon := "✓"
		if !stage.Completed {
			icon = "✗"
		}
		
		fmt.Printf("%s %-15s %s\n", icon, stage.Stage, duration.Round(time.Millisecond))
	}
	
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("Total Duration: %s\n", totalDuration.Round(time.Millisecond))
	
	if op.ignoredCount > 0 {
		fmt.Printf("\n⚠️  %d lines couldn't be parsed (saved in %s)\n", op.ignoredCount, op.debugDir)
	}
}

// stripANSI removes ANSI color codes from a string
func stripANSI(str string) string {
	// This is a simple implementation - could be improved with a proper ANSI stripping library
	// For now, we'll just handle the most common cases
	result := str
	
	// Remove color codes
	for _, code := range []string{
		"\033[0m", "\033[1m", "\033[2m", "\033[3m", "\033[4m",
		"\033[30m", "\033[31m", "\033[32m", "\033[33m", "\033[34m", "\033[35m", "\033[36m", "\033[37m",
		"\033[90m", "\033[91m", "\033[92m", "\033[93m", "\033[94m", "\033[95m", "\033[96m", "\033[97m",
	} {
		result = strings.ReplaceAll(result, code, "")
	}
	
	// Remove more complex ANSI sequences
	// This is a simplified approach - a proper regex would be better
	for strings.Contains(result, "\033[") {
		start := strings.Index(result, "\033[")
		if start == -1 {
			break
		}
		end := strings.IndexAny(result[start:], "mGKHJ")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	
	return result
}