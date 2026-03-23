package csync

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
)

// result holds the outcome of a single loader invocation.
type result[T any] struct {
	value T
	err   error
}

// LazySlice fans out loader functions across a bounded goroutine pool.
// Results are indexed positionally — Get(i) returns the result of loaders[i]
// regardless of completion order.
type LazySlice[T any] struct {
	loaders []func() (T, error)
	results []result[T]
	loaded  bool
}

// maxWorkers returns the bounded concurrency limit.
func maxWorkers() int {
	n := runtime.NumCPU()
	if n > 8 {
		n = 8
	}
	if n < 1 {
		n = 1
	}
	return n
}

// NewLazySlice creates a LazySlice. Call LoadAll() to execute loaders.
func NewLazySlice[T any](loaders []func() (T, error)) *LazySlice[T] {
	if loaders == nil {
		loaders = []func() (T, error){}
	}
	return &LazySlice[T]{
		loaders: loaders,
		results: make([]result[T], len(loaders)),
	}
}

// Len returns the number of loader slots.
func (ls *LazySlice[T]) Len() int {
	return len(ls.loaders)
}

// LoadAll executes all loaders concurrently with a bounded worker pool.
// Blocks until every loader has completed. Returns a non-nil error if
// any loader failed (individual errors retrievable via Get).
func (ls *LazySlice[T]) LoadAll() error {
	if ls.loaded || len(ls.loaders) == 0 {
		return nil
	}
	ls.loaded = true

	workers := maxWorkers()
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for i, loader := range ls.loaders {
		wg.Add(1)
		sem <- struct{}{} // acquire slot (blocks if pool is full)
		go func() {
			defer wg.Done()
			defer func() { <-sem }() // release slot

			// Panic recovery: convert panic to error.
			defer func() {
				if r := recover(); r != nil {
					ls.results[i] = result[T]{
						err: fmt.Errorf("loader %d panicked: %v", i, r),
					}
				}
			}()

			val, err := loader()
			ls.results[i] = result[T]{value: val, err: err}
		}()
	}

	wg.Wait()

	// Build aggregate error if any loaders failed.
	var failed int
	var msgs []string
	for i, r := range ls.results {
		if r.err != nil {
			failed++
			msgs = append(msgs, fmt.Sprintf("loader %d: %v", i, r.err))
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d/%d loaders failed: %s",
			failed, len(ls.loaders), strings.Join(msgs, "; "))
	}

	return nil
}

// Get returns the result at index i. Panics if i is out of range.
// Must be called after LoadAll().
func (ls *LazySlice[T]) Get(i int) (T, error) {
	return ls.results[i].value, ls.results[i].err
}

// MustGet returns the value at index i, panicking if the loader returned an error.
func (ls *LazySlice[T]) MustGet(i int) T {
	if ls.results[i].err != nil {
		panic(fmt.Sprintf("csync.LazySlice.MustGet(%d): %v", i, ls.results[i].err))
	}
	return ls.results[i].value
}
