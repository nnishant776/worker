package worker

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func withSchedulingDelay(dur time.Duration) Option {
	return workerOptFunc(func(cfg *workerConfig) {
		cfg.schedDelay = dur
	})
}

func Test_ThreadPoolWorker(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		t.Run("Without options", func(t *testing.T) {
			var maxQueueSize = runtime.GOMAXPROCS(0)
			var ctx = context.Background()
			var thpWorker = NewThreadPoolWorker(ctx)
			assert.Equal(t, maxQueueSize, thpWorker.cfg.queueSize)
			assert.Equal(t, maxQueueSize, thpWorker.cfg.poolSize)
			assert.Equal(t, true, thpWorker.cfg.autoStart)
			assert.Equal(t, true, thpWorker.cfg.autoRespawn)
			assert.NotEqual(t, nil, thpWorker.ctx)
		})

		t.Run("With options", func(t *testing.T) {
			t.Run("WithNoAutoStart", func(t *testing.T) {
				var ctx = context.Background()
				var thpWorker = NewThreadPoolWorker(ctx, WithNoAutoStart())
				assert.Equal(t, false, thpWorker.cfg.autoStart)
			})

			t.Run("WithNoWorkerAutoRespawn", func(t *testing.T) {
				var ctx = context.Background()
				var thpWorker = NewThreadPoolWorker(ctx, WithNoWorkerAutoRespawn())
				assert.Equal(t, false, thpWorker.cfg.autoRespawn)
			})

			t.Run("WithQueueSize", func(t *testing.T) {
				var ctx = context.Background()
				var maxQueueSize = runtime.GOMAXPROCS(0)
				var thpWorker = NewThreadPoolWorker(ctx, WithQueueSize(4))
				assert.Equal(t, 4, thpWorker.cfg.queueSize)
				assert.Equal(t, maxQueueSize, thpWorker.cfg.poolSize)
			})

			t.Run("WithPoolSize", func(t *testing.T) {
				var ctx = context.Background()
				var maxQueueSize = runtime.GOMAXPROCS(0)
				var thpWorker = NewThreadPoolWorker(ctx, WithPoolSize(4))
				assert.Equal(t, maxQueueSize, thpWorker.cfg.queueSize)
				assert.Equal(t, 4, thpWorker.cfg.poolSize)
			})
		})
	})

	t.Run("Task completion", func(t *testing.T) {
		var ctx = context.Background()
		var thpWorker = NewThreadPoolWorker(ctx)
		var handleMap = map[uint64]TaskHandle{}

		for i := 0; i < 20; i++ {
			var task = Task{
				Run: func() {
					// t.Logf("Finished task: id = %d", i+1)
				},
			}
			handle, err := thpWorker.Submit(ctx, task)
			if err != nil {
				// t.Logf("Failed to submit task: err = %s", err)
				continue
			}
			handleMap[handle.id] = handle
		}

		for _, handle := range handleMap {
			var ch = handle.Done()
			<-ch
			assert.Equal(t, true, handle.IsDone())
		}

		thpWorker.Shutdown()
		thpWorker.Wait()
	})

	t.Run("Task cancel", func(t *testing.T) {
		var ctx = context.Background()
		var thpWorker = NewThreadPoolWorker(ctx, withSchedulingDelay(10*time.Millisecond))
		var handleMap = map[uint64]TaskHandle{}

		for i := 0; i < 20; i++ {
			var task = Task{
				Run: func() {
					// t.Logf("Finished task: id = %d", i+1)
				},
			}
			handle, err := thpWorker.Submit(ctx, task)
			if err != nil {
				// t.Logf("Failed to submit task: err = %s", err)
				continue
			}
			handleMap[handle.id] = handle
			handle.Cancel()
		}

		for _, handle := range handleMap {
			assert.Equal(t, true, handle.IsCancelled())
		}

		thpWorker.Shutdown()
		thpWorker.Wait()
	})

	t.Run("Submit timeout", func(t *testing.T) {
		var ctx = context.Background()
		var thpWorker = NewThreadPoolWorker(ctx, WithNoAutoStart())

		for i := 0; i < 20; i++ {
			var task = Task{
				Run: func() {
					// t.Logf("Finished task: id = %d", i+1)
				},
			}
			chCtx, _ := context.WithTimeout(ctx, 10*time.Millisecond)
			_, err := thpWorker.Submit(chCtx, task)
			if err != nil {
				// t.Logf("Failed to submit task: err = %s", err)
				assert.Equal(t, true, errors.Is(err, context.DeadlineExceeded))
				continue
			}
		}

		thpWorker.Shutdown()
		thpWorker.Wait()
	})

	t.Run("Worker panic", func(t *testing.T) {
		var ctx = context.Background()
		var thpWorker = NewThreadPoolWorker(ctx)
		var handleMap = map[uint64]TaskHandle{}

		for i := 0; i < 20; i++ {
			var task = Task{
				Run: func() {
					panic(fmt.Sprintf("Task paniced: id = %d", i+1))
				},
			}
			handle, err := thpWorker.Submit(ctx, task)
			if err != nil {
				t.Logf("Failed to submit task: err = %s", err)
				continue
			}
			handleMap[handle.id] = handle
		}

		time.Sleep(1 * time.Second)

		thpWorker.Shutdown()
		thpWorker.Wait()

		for _, handle := range handleMap {
			assert.NotEqual(t, nil, handle.Panic(), "Panic not observed for: id = %d", handle.id)
		}
	})

	t.Run("Producer", func(t *testing.T) {
		t.Run("Manual start", func(t *testing.T) {
			var ctx = context.Background()
			var thpWorker = NewThreadPoolWorker(ctx, WithNoAutoStart())
			var handleMap = map[uint64]TaskHandle{}

			assert.Equal(t, true, thpWorker.stop == nil)

			var doneCh = make(chan struct{})

			go func() {
				for i := 0; i < 20; i++ {
					var task = Task{
						Run: func() {
							// t.Logf("Finished task: id = %d", i+1)
						},
					}
					handle, err := thpWorker.Submit(ctx, task)
					if err != nil {
						// t.Logf("Failed to submit task: err = %s", err)
						continue
					}
					handleMap[handle.id] = handle
				}

				doneCh <- struct{}{}
			}()

			thpWorker.Start()

			<-doneCh

			for _, handle := range handleMap {
				var ch = handle.Done()
				<-ch
				assert.Equal(t, true, handle.IsDone())
			}

			thpWorker.Shutdown()
			thpWorker.Wait()

		})

		t.Run("Manual stop", func(t *testing.T) {
			var ctx = context.Background()
			var thpWorker = NewThreadPoolWorker(ctx, withSchedulingDelay(10*time.Millisecond))
			var handleMap = map[uint64]TaskHandle{}

			assert.Equal(t, false, thpWorker.stop == nil)

			var doneCh = make(chan struct{})

			go func() {
				for i := 0; i < 20; i++ {
					var task = Task{
						Run: func() {
							// t.Logf("Finished task: id = %d", i+1)
						},
					}
					handle, err := thpWorker.Submit(ctx, task)
					if err != nil {
						// t.Logf("Failed to submit task: err = %s", err)
						continue
					}
					handleMap[handle.id] = handle
				}

				doneCh <- struct{}{}
			}()

			thpWorker.Stop()

			assert.Equal(t, true, thpWorker.stop == nil)

			thpWorker.Start()

			<-doneCh

			for _, handle := range handleMap {
				var ch = handle.Done()
				<-ch
				assert.Equal(t, true, handle.IsDone())
			}

			thpWorker.Shutdown()
			thpWorker.Wait()
		})
	})
}
