package concurrency

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolRunsAllSubmittedTasks(t *testing.T) {
	scenarios := []struct {
		name      string
		workers   int
		queueSize int
		producers int
		perProd   int
	}{
		{name: "single worker small queue", workers: 1, queueSize: 1, producers: 4, perProd: 50},
		{name: "few workers", workers: 4, queueSize: 16, producers: 8, perProd: 125},
		{name: "many workers wide queue", workers: 16, queueSize: 256, producers: 32, perProd: 100},
		{name: "unbuffered queue", workers: 8, queueSize: 0, producers: 16, perProd: 64},
	}

	for _, sc := range scenarios {
		sc := sc
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()

			pool := NewPool(sc.workers, sc.queueSize)
			pool.Start()

			var counter int64
			expected := int64(sc.producers * sc.perProd)

			var producers sync.WaitGroup
			producers.Add(sc.producers)
			for i := 0; i < sc.producers; i++ {
				go func() {
					defer producers.Done()
					for j := 0; j < sc.perProd; j++ {
						err := pool.Submit(func() {
							atomic.AddInt64(&counter, 1)
						})
						require.NoError(t, err)
					}
				}()
			}

			producers.Wait()
			pool.Stop()

			assert.Equal(t, expected, atomic.LoadInt64(&counter))
		})
	}
}

func TestSubmitAfterStopReturnsError(t *testing.T) {
	pool := NewPool(2, 4)
	pool.Start()
	pool.Stop()

	err := pool.Submit(func() {})
	assert.ErrorIs(t, err, ErrPoolStopped)
}

func TestTrySubmitReturnsFalseWhenStopped(t *testing.T) {
	pool := NewPool(2, 4)
	pool.Start()
	pool.Stop()

	ok := pool.TrySubmit(func() {})
	assert.False(t, ok)
}

func TestTrySubmitReturnsFalseWhenFull(t *testing.T) {
	pool := NewPool(1, 1)
	pool.Start()

	release := make(chan struct{})
	occupied := make(chan struct{})

	require.True(t, pool.TrySubmit(func() {
		close(occupied)
		<-release
	}))
	<-occupied

	require.True(t, pool.TrySubmit(func() {}))

	full := false
	for i := 0; i < 1000; i++ {
		if !pool.TrySubmit(func() {}) {
			full = true
			break
		}
	}
	assert.True(t, full)

	close(release)
	pool.Stop()
}

func TestStopIsIdempotent(t *testing.T) {
	pool := NewPool(4, 8)
	pool.Start()

	var counter int64
	for i := 0; i < 8; i++ {
		require.NoError(t, pool.Submit(func() {
			atomic.AddInt64(&counter, 1)
		}))
	}

	var stoppers sync.WaitGroup
	stoppers.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer stoppers.Done()
			pool.Stop()
		}()
	}
	stoppers.Wait()

	assert.Equal(t, int64(8), atomic.LoadInt64(&counter))
	assert.Equal(t, 0, pool.QueueLen())
}

func TestStopDrainsQueuedTasks(t *testing.T) {
	pool := NewPool(2, 64)
	pool.Start()

	var counter int64
	const n = 64
	for i := 0; i < n; i++ {
		require.NoError(t, pool.Submit(func() {
			time.Sleep(time.Millisecond)
			atomic.AddInt64(&counter, 1)
		}))
	}

	pool.Stop()
	assert.Equal(t, int64(n), atomic.LoadInt64(&counter))
}

func TestQueueLenReflectsBufferedTasks(t *testing.T) {
	pool := NewPool(1, 4)
	pool.Start()

	release := make(chan struct{})
	occupied := make(chan struct{})
	require.True(t, pool.TrySubmit(func() {
		close(occupied)
		<-release
	}))
	<-occupied

	require.True(t, pool.TrySubmit(func() {}))
	require.True(t, pool.TrySubmit(func() {}))
	assert.Equal(t, 2, pool.QueueLen())

	close(release)
	pool.Stop()
}
