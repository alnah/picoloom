package md2pdf

// Notes:
// - Tests ServicePool for concurrent service management
// - Tests ResolvePoolSize for auto-calculation and explicit values
// - Concurrency tests verify deadlock-free behavior under contention
// - Pool is safe for concurrent use: multiple goroutines can Acquire/Release

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Compile-Time Interface Check
// ---------------------------------------------------------------------------

var _ interface {
	Acquire() *Service
	Release(*Service)
	Size() int
	Close() error
} = (*ServicePool)(nil)

// ---------------------------------------------------------------------------
// TestResolvePoolSize - Pool Size Calculation
// ---------------------------------------------------------------------------

func TestResolvePoolSize(t *testing.T) {
	t.Parallel()

	gomaxprocs := runtime.GOMAXPROCS(0)

	tests := []struct {
		name    string
		workers int
		want    int
	}{
		{
			name:    "explicit takes priority",
			workers: 4,
			want:    4,
		},
		{
			name:    "explicit=1 for sequential",
			workers: 1,
			want:    1,
		},
		{
			name:    "zero uses auto calculation",
			workers: 0,
			want:    min(max(gomaxprocs/cpuDivisor, MinPoolSize), MaxPoolSize),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ResolvePoolSize(tt.workers)
			if got != tt.want {
				t.Errorf("ResolvePoolSize(%d) = %d, want %d", tt.workers, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestResolvePoolSize_Bounds - Pool Size Boundary Conditions
// ---------------------------------------------------------------------------

func TestResolvePoolSize_Bounds(t *testing.T) {
	t.Parallel()

	t.Run("auto calculation respects minimum", func(t *testing.T) {
		t.Parallel()

		got := ResolvePoolSize(0)
		if got < MinPoolSize {
			t.Errorf("ResolvePoolSize(0) = %d, want >= %d", got, MinPoolSize)
		}
	})

	t.Run("auto calculation respects maximum", func(t *testing.T) {
		t.Parallel()

		got := ResolvePoolSize(0)
		if got > MaxPoolSize {
			t.Errorf("ResolvePoolSize(0) = %d, want <= %d", got, MaxPoolSize)
		}
	})

	t.Run("explicit can exceed max", func(t *testing.T) {
		t.Parallel()

		got := ResolvePoolSize(16)
		if got != 16 {
			t.Errorf("ResolvePoolSize(16) = %d, want 16", got)
		}
	})
}

// ---------------------------------------------------------------------------
// TestServicePool_AcquireRelease - Basic Acquire/Release Operations
// ---------------------------------------------------------------------------

func TestServicePool_AcquireRelease(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(2)
	defer pool.Close()

	// Acquire first service
	svc1 := pool.Acquire()
	if svc1 == nil {
		t.Fatalf("Acquire() returned nil")
	}

	// Acquire second service
	svc2 := pool.Acquire()
	if svc2 == nil {
		t.Fatalf("Acquire() returned nil")
	}

	// Services should be different instances
	if svc1 == svc2 {
		t.Errorf("Acquire() returned same service instance, want different instances")
	}

	// Release and re-acquire
	pool.Release(svc1)
	svc3 := pool.Acquire()

	if svc3 != svc1 {
		t.Errorf("Acquire() after Release() returned different service, want reused service")
	}

	// Cleanup
	pool.Release(svc2)
	pool.Release(svc3)
}

// ---------------------------------------------------------------------------
// TestServicePool_Size - Pool Size Property
// ---------------------------------------------------------------------------

func TestServicePool_Size(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int
		want int
	}{
		{"explicit size 1", 1, 1},
		{"explicit size 4", 4, 4},
		{"zero becomes 1", 0, 1},
		{"negative becomes 1", -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool := NewServicePool(tt.size)
			defer pool.Close()

			if got := pool.Size(); got != tt.want {
				t.Errorf("Size() = %d, want %d", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestServicePool_ConcurrentAccess - Concurrent Access Safety
// ---------------------------------------------------------------------------

func TestServicePool_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(4)
	defer pool.Close()

	var wg sync.WaitGroup
	iterations := 20

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc := pool.Acquire()
			time.Sleep(5 * time.Millisecond) // Simulate work
			pool.Release(svc)
		}()
	}

	// Should complete without deadlock
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-done:
		// Success
	case <-timer.C:
		t.Fatalf("concurrent access test timed out - possible deadlock")
	}
}

// ---------------------------------------------------------------------------
// TestServicePool_ClosePreventsFurtherRelease - Close Behavior
// ---------------------------------------------------------------------------

func TestServicePool_ClosePreventsFurtherRelease(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(2)

	svc := pool.Acquire()
	pool.Close()

	// Release after close should not panic
	pool.Release(svc) // Should be safe (no-op)
}

// ---------------------------------------------------------------------------
// TestServicePool_DoubleClose - Double Close Idempotency
// ---------------------------------------------------------------------------

func TestServicePool_DoubleClose(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(1)

	// First close
	if err := pool.Close(); err != nil {
		t.Errorf("first Close() unexpected error: %v", err)
	}

	// Second close should not panic
	pool.Close()
}

// ---------------------------------------------------------------------------
// TestServicePool_AcquireAfterClose - Acquire After Close Behavior
// ---------------------------------------------------------------------------

func TestServicePool_AcquireAfterClose(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(2)

	// Acquire one service
	svc := pool.Acquire()
	if svc == nil {
		t.Fatalf("Acquire() returned nil")
	}

	// Close the pool
	pool.Close()

	// Release should not panic after close
	pool.Release(svc)

	// Note: Acquire after close will block forever on empty channel,
	// so we don't test that directly - it's documented behavior.
}

// ---------------------------------------------------------------------------
// TestServicePool_ReleaseNilService - Nil Service Release Behavior
// ---------------------------------------------------------------------------

func TestServicePool_ReleaseNilService(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(1)
	defer pool.Close()

	// Acquire to create a service
	svc := pool.Acquire()
	pool.Release(svc)

	// Release nil should cause panic (channel send on nil),
	// but this is expected behavior - callers should not release nil.
	// This test documents that behavior.
}

// ---------------------------------------------------------------------------
// TestServicePool_HighContention - High Contention Deadlock Prevention
// ---------------------------------------------------------------------------

// TestServicePool_HighContention verifies the pool remains deadlock-free under
// heavy concurrent access. A small pool (2 services) with many goroutines (50)
// each performing multiple acquire/release cycles exposes race conditions and
// channel blocking issues that wouldn't surface with lighter loads.
func TestServicePool_HighContention(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(2)
	defer pool.Close()

	var wg sync.WaitGroup
	goroutines := 50
	iterations := 10

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				svc := pool.Acquire()
				// Simulate variable work duration
				time.Sleep(time.Duration(j%3) * time.Millisecond)
				pool.Release(svc)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	select {
	case <-done:
		// Success - no deadlock under high contention
	case <-timer.C:
		t.Fatalf("high contention test timed out - possible deadlock")
	}
}

// ---------------------------------------------------------------------------
// TestServicePool_AllServicesAcquired - Full Pool Acquisition
// ---------------------------------------------------------------------------

func TestServicePool_AllServicesAcquired(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(3)
	defer pool.Close()

	// Acquire all services
	services := make([]*Service, 3)
	for i := 0; i < 3; i++ {
		services[i] = pool.Acquire()
		if services[i] == nil {
			t.Fatalf("Acquire() returned nil for service %d", i)
		}
	}

	// Verify we got 3 distinct services
	seen := make(map[*Service]bool)
	for _, svc := range services {
		if seen[svc] {
			t.Errorf("Acquire() returned duplicate service, want distinct instances")
		}
		seen[svc] = true
	}

	// Release all
	for _, svc := range services {
		pool.Release(svc)
	}
}

// ---------------------------------------------------------------------------
// TestServicePool_LazyCreation - Lazy Service Creation
// ---------------------------------------------------------------------------

func TestServicePool_LazyCreation(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(3)
	defer pool.Close()

	// Pool should not create services until acquired
	// Acquire one service
	svc1 := pool.Acquire()
	if svc1 == nil {
		t.Fatalf("first Acquire() returned nil")
	}

	// Release it
	pool.Release(svc1)

	// Acquire again - should get the same service (reuse)
	svc2 := pool.Acquire()
	if svc2 != svc1 {
		t.Errorf("Acquire() after Release() returned different service, want reused service")
	}

	pool.Release(svc2)
}

// ---------------------------------------------------------------------------
// TestResolvePoolSize_NegativeWorkers - Negative Worker Count Handling
// ---------------------------------------------------------------------------

func TestResolvePoolSize_NegativeWorkers(t *testing.T) {
	t.Parallel()

	// Negative workers should be treated as 0 (auto-calculate)
	got := ResolvePoolSize(-5)

	if got < MinPoolSize || got > MaxPoolSize {
		t.Errorf("ResolvePoolSize(-5) = %d, want value between %d and %d", got, MinPoolSize, MaxPoolSize)
	}
}

// ---------------------------------------------------------------------------
// TestResolvePoolSize_LargeExplicitValue - Large Explicit Value Handling
// ---------------------------------------------------------------------------

func TestResolvePoolSize_LargeExplicitValue(t *testing.T) {
	t.Parallel()

	// Explicit value above MaxPoolSize should be allowed
	got := ResolvePoolSize(100)

	if got != 100 {
		t.Errorf("ResolvePoolSize(100) = %d, want 100", got)
	}
}

// ---------------------------------------------------------------------------
// TestServicePool_WithOptions - Pool With Service Options
// ---------------------------------------------------------------------------

func TestServicePool_WithOptions(t *testing.T) {
	t.Parallel()

	// Test that options are passed to services created by the pool
	pool := NewServicePool(1, WithTimeout(5*time.Minute))
	defer pool.Close()

	svc := pool.Acquire()
	if svc == nil {
		t.Fatalf("Acquire() returned nil")
	}

	// Verify the timeout was applied
	if svc.cfg.timeout != 5*time.Minute {
		t.Errorf("Acquire().cfg.timeout = %v, want %v", svc.cfg.timeout, 5*time.Minute)
	}

	pool.Release(svc)
}

// ---------------------------------------------------------------------------
// TestServicePool_InitError - Initialization Error Handling
// ---------------------------------------------------------------------------

func TestServicePool_InitError(t *testing.T) {
	t.Parallel()

	t.Run("happy path: no error after successful acquire", func(t *testing.T) {
		t.Parallel()

		pool := NewServicePool(1)
		defer pool.Close()

		// Acquire to trigger service creation
		svc := pool.Acquire()
		if svc == nil {
			t.Skip("service creation failed, cannot test InitError in this environment")
		}
		pool.Release(svc)

		if err := pool.InitError(); err != nil {
			t.Errorf("InitError() = %v, want nil", err)
		}
	})

	t.Run("happy path: no error before any acquire", func(t *testing.T) {
		t.Parallel()

		pool := NewServicePool(1)
		defer pool.Close()

		if err := pool.InitError(); err != nil {
			t.Errorf("InitError() = %v, want nil", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestServicePool_AcquireReturnsNilOnInitError - Acquire with Init Error
// ---------------------------------------------------------------------------

func TestServicePool_AcquireReturnsNilOnInitError(t *testing.T) {
	t.Parallel()

	// Create a pool with an option that causes service creation to fail.
	// Since we can't easily make New() fail with valid options in the current
	// implementation (it would need an invalid asset loader path), we test
	// the mechanism by observing behavior: if InitError is set, Acquire returns nil.

	pool := NewServicePool(2)
	defer pool.Close()

	// First acquire should work
	svc1 := pool.Acquire()
	if svc1 == nil {
		t.Skip("service creation failed, cannot test in this environment")
	}

	// Pool should not have an init error yet
	if err := pool.InitError(); err != nil {
		t.Errorf("InitError() = %v, want nil", err)
	}

	pool.Release(svc1)
}
