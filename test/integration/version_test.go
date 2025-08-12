package integration

import (
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"strings"
	"testing"
	
)

func TestVersionCommand(t *testing.T) {
	// Test v1
	helpers.IsolatedTestWithVersion(t, "v1", helpers.BinaryV1, func(t *testing.T, ctx *helpers.TrebContext) {
		output, err := ctx.Treb("version")
		if err != nil {
			t.Fatalf("version command failed: %v\nOutput: %s", err, output)
		}
		if !strings.Contains(output, "treb v") {
			t.Errorf("Expected 'treb v' in output, got: %s", output)
		}
	})
	
	// Test v2 if available
	if helpers.V2BinaryExists() {
		helpers.IsolatedTestWithVersion(t, "v2", helpers.BinaryV2, func(t *testing.T, ctx *helpers.TrebContext) {
			output, err := ctx.Treb("version")
			if err != nil {
				t.Fatalf("version command failed: %v\nOutput: %s", err, output)
			}
			if !strings.Contains(output, "treb version") {
				t.Errorf("Expected 'treb version' in output, got: %s", output)
			}
		})
	}
}