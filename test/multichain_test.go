package integration_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestMultichainDeployment deploys the same contract to two local anvil instances
// and verifies that the addresses are deterministic and identical across chains.
func TestMultichainDeployment(t *testing.T) {
	ctx0 := NewTrebContext(t).WithNetwork("anvil0")
	ctx1 := NewTrebContext(t).WithNetwork("anvil1")

	// Generate deployment script once
	if out, err := ctx0.treb("gen", "deploy", "src/Counter.sol:Counter"); err != nil {
		t.Fatalf("failed to generate deploy script: %v\nOutput:\n%s", err, out)
	}

	// Deploy to anvil0
	if out, err := ctx0.treb("run", "script/deploy/DeployCounter.s.sol"); err != nil {
		t.Fatalf("failed to deploy to anvil0: %v\nOutput:\n%s", err, out)
	}

	// Deploy to anvil1
	if out, err := ctx1.treb("run", "script/deploy/DeployCounter.s.sol"); err != nil {
		t.Fatalf("failed to deploy to anvil1: %v\nOutput:\n%s", err, out)
	}

	// Extract addresses via show command and compare
	addr0 := mustExtractAddress(t, ctx0)
	addr1 := mustExtractAddress(t, ctx1)

	if !strings.EqualFold(addr0, addr1) {
		t.Fatalf("expected same address across networks, got %s (anvil0) vs %s (anvil1)", addr0, addr1)
	}
}

func mustExtractAddress(t *testing.T, ctx *TrebContext) string {
	out, err := ctx.treb("show", "Counter")
	if err != nil {
		t.Fatalf("failed to show deployment on %s: %v\nOutput:\n%s", ctx.network, err, out)
	}
	// Lines typically contain: "Address: 0x..."
	re := regexp.MustCompile(`(?i)Address:\s*(0x[0-9a-fA-F]{40})`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		t.Fatalf("could not find address in show output for %s:\n%s", ctx.network, out)
	}
	return m[1]
}
