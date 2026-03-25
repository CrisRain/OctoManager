// Package logbatch provides an async, batch log writer.
// Callers enqueue individual entries via Add; a background goroutine
// accumulates them and flushes in batches to reduce database round-trips.
package logbatch

import (
	"context"
	"sync"
	"time"
)

const (
	defaultFlushInterval = 100 * time.Millisecond
	defaultMaxBatch      = 50
	defaultChannelCap    = 512
)

// Batcher buffers T entries and periodically flushes them via a user-supplied function.
// It is safe for concurrent use.
type Batcher[T any] struct {
	ch       chan T
	flushFn  func(context.Context, []T) error
	interval time.Duration
	maxBatch int
	wg       sync.WaitGroup
}

// New creates a Batcher with default settings (flush every 100 ms or every 50 entries).
func New[T any](flushFn func(context.Context, []T) error) *Batcher[T] {
	return &Batcher[T]{
		ch:       make(chan T, defaultChannelCap),
		flushFn:  flushFn,
		interval: defaultFlushInterval,
		maxBatch: defaultMaxBatch,
	}
}

// Add enqueues an entry. If the internal buffer is full the entry is dropped
// rather than blocking the caller.
func (b *Batcher[T]) Add(v T) {
	select {
	case b.ch <- v:
	default:
	}
}

// Run processes entries until ctx is cancelled, then drains the channel
// and flushes any remaining entries before returning.
// Call this in a separate goroutine; use Wait to block until it finishes.
func (b *Batcher[T]) Run(ctx context.Context) {
	b.wg.Add(1)
	defer b.wg.Done()

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	batch := make([]T, 0, b.maxBatch)

	for {
		select {
		case v := <-b.ch:
			batch = append(batch, v)
			if len(batch) >= b.maxBatch {
				b.doFlush(ctx, &batch)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				b.doFlush(ctx, &batch)
			}

		case <-ctx.Done():
			// Drain the channel before exiting.
			for {
				select {
				case v := <-b.ch:
					batch = append(batch, v)
					if len(batch) >= b.maxBatch {
						b.doFlush(context.Background(), &batch)
					}
				default:
					if len(batch) > 0 {
						b.doFlush(context.Background(), &batch)
					}
					return
				}
			}
		}
	}
}

// Wait blocks until the Run goroutine has finished (i.e. all pending entries flushed).
func (b *Batcher[T]) Wait() {
	b.wg.Wait()
}

func (b *Batcher[T]) doFlush(ctx context.Context, batch *[]T) {
	_ = b.flushFn(ctx, *batch)
	*batch = (*batch)[:0]
}
