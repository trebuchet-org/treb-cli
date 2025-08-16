package render

import (
	"fmt"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// GenerateRenderer renders init command results
type GenerateRenderer struct{}

// NewInitRenderer creates a new init renderer
func NewGenerateRenderer() Renderer[*usecase.GenerateScriptResult] {
	return &GenerateRenderer{}
}

func (r *GenerateRenderer) Render(result *usecase.GenerateScriptResult) error {
	// Display result
	fmt.Printf("\n✅ Generated deployment script: %s\n", result.ScriptPath)
	for _, instruction := range result.Instructions {
		fmt.Println(instruction)
	}
	return nil
}
