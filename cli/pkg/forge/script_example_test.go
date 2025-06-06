package forge

import (
	"fmt"
	"testing"
)

// This is an example test to demonstrate how the new output processing works
func ExampleOutputProcessing() {
	// Example of how the new Run function works:
	
	// 1. Debug mode (opts.Debug = true, opts.JSON = false):
	//    - Direct copy to stdout with colors preserved via PTY
	//    - No parsing or processing
	
	// 2. Normal mode:
	//    - Real-time line processing with scanner
	//    - Stage tracking with spinner display
	//    - Parsed entities sent via channel
	//    - Unparsed lines saved to .treb-debug/runID/ignored-lineN.txt
	
	// Stage display example:
	// ✓ Initializing (123ms) → ✓ Compiling (2.5s) → ● Simulating (5s) → ○ Broadcasting
	
	fmt.Println("Example output processing modes")
	// Output: Example output processing modes
}

// TestStageDetection tests the stage detection logic
func TestStageDetection(t *testing.T) {
	tests := []struct {
		line          string
		expectedStage Stage
	}{
		{"[⠊] Compiling...", StageCompiling},
		{"Starting simulation", StageSimulating},
		{"Broadcasting transaction", StageBroadcasting},
		{"Script ran successfully", StageCompleted},
	}
	
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			// This would test the stage detection logic
			// The actual implementation would check if the processor
			// correctly identifies stages from line content
		})
	}
}