package helpers

import (
	"testing"
)

// IsolatedTest runs a test with full isolation
func IsolatedTest(t *testing.T, name string, fn func(t *testing.T, ctx *TestContext)) {
	t.Run(name, func(t *testing.T) {
		t.Parallel()

		pool := GetGlobalPool()
		if pool == nil {
			t.Fatal("Test pool not initialized")
		}
		testCtx := pool.Acquire(t)

		defer func() {
			if err := pool.Release(testCtx); err != nil {
				t.Logf("Warning: failed to release test context: %v", err)
			}
		}()

		// Run the test
		fn(t, testCtx)
	})
}
