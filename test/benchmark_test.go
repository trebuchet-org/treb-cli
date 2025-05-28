package integration_test

import (
	"os/exec"
	"testing"
)

// Benchmark example
func BenchmarkVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(trebBin, "version")
		cmd.Run()
	}
}