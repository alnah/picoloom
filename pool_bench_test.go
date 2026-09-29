//go:build bench

package picoloom

// Notes:
// - Benchmarks for ServicePool performance
// - Tests pool size calculation, acquire/release cycle, and contention scenarios
// - Uses pre-warming to avoid measuring service creation overhead
// - Tests parallel access patterns with various goroutine counts

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

func BenchmarkResolvePoolSize(b *testing.B) {
	workers := []int{0, 1, 2, 4, 8}

	for _, w := range workers {
		b.Run(workerName(w), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := ResolvePoolSize(w)
				_ = result
			}
		})
	}
}

func workerName(w int) string {
	if w == 0 {
		return "auto"
	}
	return fmt.Sprintf("%d", w)
}

func BenchmarkServicePoolAcquireRelease(b *testing.B) {
	sizes := []int{1, 2, 4, 8}

	for _, size := range sizes {
		b.Run(poolSizeName(size), func(b *testing.B) {
			pool := NewServicePool(size)
			// Pre-warm the pool by acquiring and releasing all slots
			services := make([]*Service, size)
			for i := 0; i < size; i++ {
				services[i] = pool.Acquire()
			}
			for i := 0; i < size; i++ {
				pool.Release(services[i])
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				svc := pool.Acquire()
				pool.Release(svc)
			}

			b.StopTimer()
			pool.Close()
		})
	}
}

func poolSizeName(size int) string {
	return fmt.Sprintf("size_%d", size)
}

func BenchmarkServicePoolContention(b *testing.B) {
	poolSize := 4
	goroutines := []int{4, 8, 16, 32}

	for _, g := range goroutines {
		b.Run(goroutineName(g), func(b *testing.B) {
			pool := NewServicePool(poolSize)
			// Pre-warm
			services := make([]*Service, poolSize)
			for i := 0; i < poolSize; i++ {
				services[i] = pool.Acquire()
			}
			for i := 0; i < poolSize; i++ {
				pool.Release(services[i])
			}

			b.ReportAllocs()
			b.ResetTimer()

			var wg sync.WaitGroup
			opsPerGoroutine := b.N / g
			if opsPerGoroutine < 1 {
				opsPerGoroutine = 1
			}

			for i := 0; i < g; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < opsPerGoroutine; j++ {
						svc := pool.Acquire()
						// Simulate minimal work
						runtime.Gosched()
						pool.Release(svc)
					}
				}()
			}
			wg.Wait()

			b.StopTimer()
			pool.Close()
		})
	}
}

func goroutineName(g int) string {
	return fmt.Sprintf("goroutines_%d", g)
}

func BenchmarkServicePoolParallel(b *testing.B) {
	pool := NewServicePool(runtime.GOMAXPROCS(0))
	// Pre-warm
	size := pool.Size()
	services := make([]*Service, size)
	for i := 0; i < size; i++ {
		services[i] = pool.Acquire()
	}
	for i := 0; i < size; i++ {
		pool.Release(services[i])
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			svc := pool.Acquire()
			pool.Release(svc)
		}
	})

	b.StopTimer()
	pool.Close()
}

func BenchmarkNewServicePool(b *testing.B) {
	sizes := []int{1, 4, 8}

	for _, size := range sizes {
		b.Run(poolSizeName(size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				pool := NewServicePool(size)
				_ = pool
				// Don't close to avoid browser cleanup overhead
			}
		})
	}
}
