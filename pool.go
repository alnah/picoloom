package md2pdf

import (
	"errors"
	"runtime"
	"sync"
)

// Pool sizing constants.
const (
	// MinPoolSize ensures at least one worker is available.
	MinPoolSize = 1

	// MaxPoolSize caps browser instances to limit memory (~200MB each).
	MaxPoolSize = 8

	// cpuDivisor leaves headroom for Chrome child processes.
	cpuDivisor = 2
)

// ConverterPool manages a pool of Converter instances for parallel processing.
// Each converter has its own browser instance, enabling true parallelism.
// Converters are created lazily on first acquire to avoid startup delay.
type ConverterPool struct {
	size       int
	opts       []Option
	converters []*Converter
	sem        chan *Converter
	closedCh   chan struct{}
	mu         sync.Mutex
	created    int
	closed     bool
	initErr    error // First error encountered during converter creation
}

// ServicePool is an alias for ConverterPool for backward compatibility.
//
// Deprecated: Use ConverterPool instead. This alias will be removed in v2.
type ServicePool = ConverterPool

// NewConverterPool creates a pool with capacity for n Converter instances.
// Converters are created lazily when acquired, not at pool creation.
// Options are applied to each converter when created.
func NewConverterPool(n int, opts ...Option) *ConverterPool {
	if n < 1 {
		n = 1
	}

	return &ConverterPool{
		size:       n,
		opts:       opts,
		converters: make([]*Converter, 0, n),
		sem:        make(chan *Converter, n),
		closedCh:   make(chan struct{}),
	}
}

// NewServicePool creates a pool with capacity for n Converter instances.
//
// Deprecated: Use NewConverterPool instead. NewServicePool will be removed in v2.
func NewServicePool(n int, opts ...Option) *ConverterPool {
	return NewConverterPool(n, opts...)
}

// Acquire gets a converter from the pool, creating one if needed.
// Blocks if all converters are in use.
// Returns nil and sets internal error if converter creation fails.
// Use InitError() to check for initialization failures.
func (p *ConverterPool) Acquire() *Converter {
	// Try to get an existing converter (non-blocking)
	select {
	case conv := <-p.sem:
		return conv
	default:
	}

	// Check if we can create a new converter
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	if p.initErr != nil {
		p.mu.Unlock()
		return nil
	}
	if p.created < p.size {
		p.created++
		p.mu.Unlock()

		// Create new converter outside the lock
		conv, err := NewConverter(p.opts...)
		if err != nil {
			p.mu.Lock()
			if p.initErr == nil {
				p.initErr = err
			}
			p.created--
			p.mu.Unlock()
			return nil
		}

		p.mu.Lock()
		p.converters = append(p.converters, conv)
		p.mu.Unlock()

		return conv
	}
	p.mu.Unlock()

	// All converters created, wait for one to be released.
	// Return nil if the pool is closed while waiting.
	select {
	case conv := <-p.sem:
		return conv
	case <-p.closedCh:
		return nil
	}
}

// InitError returns the first error encountered during converter creation.
// Returns nil if all converters were created successfully.
func (p *ConverterPool) InitError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.initErr
}

// Release returns a converter to the pool.
// The lock is released before sending to avoid deadlock when channel is full.
func (p *ConverterPool) Release(conv *Converter) {
	if conv == nil {
		return
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	select {
	case p.sem <- conv:
	case <-p.closedCh:
	}
}

// Close releases all browser resources.
// Returns an aggregated error if multiple converters fail to close.
func (p *ConverterPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	close(p.closedCh)
	converters := p.converters
	p.mu.Unlock()

	var errs []error
	for _, conv := range converters {
		if err := conv.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Size returns the pool capacity.
func (p *ConverterPool) Size() int {
	return p.size
}

// ResolvePoolSize determines the optimal pool size.
// Priority: explicit workers > GOMAXPROCS-based calculation.
// Exported for use by servers and CLIs.
func ResolvePoolSize(workers int) int {
	// Explicit value takes priority
	if workers > 0 {
		return workers
	}

	// Auto-calculate based on GOMAXPROCS (adjusted by automaxprocs for containers)
	available := runtime.GOMAXPROCS(0)
	n := available / cpuDivisor

	if n < MinPoolSize {
		return MinPoolSize
	}
	if n > MaxPoolSize {
		return MaxPoolSize
	}
	return n
}
