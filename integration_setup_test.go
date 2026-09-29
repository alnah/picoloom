//go:build integration

package picoloom

// Notes:
// - Integration test setup: shared ServicePool for all integration tests
// - testPool is initialized in TestMain and closed after all tests complete
// - acquireService helper provides automatic cleanup via t.Cleanup()
// - Pool size is capped at 4 for CI environments to avoid resource exhaustion

import (
	"os"
	"testing"
	"time"
)

// testTimeout is the standard timeout for integration test operations.
const testTimeout = 30 * time.Second

// testPool is the shared ServicePool for all integration tests.
// It is initialized in TestMain and closed after all tests complete.
// Safe for concurrent use: tests only Acquire/Release, never modify the pool.
var testPool *ServicePool

func TestMain(m *testing.M) {
	// Create pool with auto-sized capacity based on CPU cores.
	// Use a conservative size for CI environments.
	poolSize := ResolvePoolSize(0)
	if poolSize > 4 {
		poolSize = 4 // Cap at 4 to avoid resource exhaustion in CI
	}

	testPool = NewServicePool(poolSize)

	code := m.Run()

	// Cleanup all browser instances
	testPool.Close()
	os.Exit(code)
}

// Helpers

// acquireService gets a service from the shared pool with automatic cleanup.
// Uses t.Cleanup() to ensure Release is called even if test panics.
func acquireService(t *testing.T) *Service {
	t.Helper()
	svc := testPool.Acquire()
	t.Cleanup(func() { testPool.Release(svc) })
	return svc
}
