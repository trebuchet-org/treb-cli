package helpers

import (
	"testing"
)

// IsolatedTest runs a test with full isolation
func IsolatedTest(t *testing.T, name string, fn func(t *testing.T, ctx *TrebContext)) {
	t.Run(name, func(t *testing.T) {
		// Always use pool-based isolation for consistency
		pool := GetGlobalPool()
		if pool == nil {
			t.Fatal("Test pool not initialized")
		}
		testCtx := pool.Acquire(t)

		// Only release if not skipping cleanup
		if !ShouldSkipCleanup() {
			defer pool.Release(testCtx)
		} else {
			defer func() {
				t.Logf("🔍 Test context not released due to skip cleanup flag: %s", testCtx.WorkDir)
			}()
		}

		// Run the test
		fn(t, testCtx.TrebContext)
	})
}
