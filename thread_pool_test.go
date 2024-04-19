package worker

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func withSchedulingDelay(dur time.Duration) WorkerOption {
	return func(cfg workerConfig) workerConfig {
		cfg.schedDelay = dur
		return cfg
	}
}

func withAllNotifier(statuses ...TaskStatus) TaskOption {
	return func(cfg taskConfig) taskConfig {
		cfg.notifier = make(chan struct{}, _TASK_STATUS_MAX_)
		for i := range _TASK_STATUS_MAX_ {
			cfg.taskBitmap[i] = true
		}
		return cfg
	}
}

type taskInfo struct {
	task     *TaskHandle
	listener <-chan struct{}
}

func Test_ThreadPoolWorker(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		t.Run("Without options", func(t *testing.T) {
			var maxQueueSize = uint32(runtime.GOMAXPROCS(0))
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
				var maxQueueSize = DefaultQueueSize()
				var thpWorker = NewThreadPoolWorker(ctx, WithQueueSize(4))
				assert.Equal(t, uint32(4), thpWorker.cfg.queueSize)
				assert.Equal(t, maxQueueSize, thpWorker.cfg.poolSize)
			})

			t.Run("WithPoolSize", func(t *testing.T) {
				var ctx = context.Background()
				var maxQueueSize = DefaultPoolSize()
				var thpWorker = NewThreadPoolWorker(ctx, WithPoolSize(4))
				assert.Equal(t, maxQueueSize, thpWorker.cfg.queueSize)
				assert.Equal(t, uint32(4), thpWorker.cfg.poolSize)
			})
		})
	})

	t.Run("Task completion", func(t *testing.T) {
		var ctx = context.Background()
		var thpWorker = NewThreadPoolWorker(ctx)
		var handleMap = map[uint64]taskInfo{}

		for i := 0; i < 20; i++ {
			var task = NewTask(func(_ uint64, _ string, _ uint32) {}, withAllNotifier(), WithHighPriority(), WithUUID(""))
			handle, err := thpWorker.Submit(ctx, task)
			if err != nil {
				// t.Logf("Failed to submit task: err = %s", err)
				continue
			}
			ch, _ := handle.Channel()
			handleMap[handle.ID()] = taskInfo{
				task:     handle,
				listener: ch,
			}
		}

		for _, handle := range handleMap {
			for {
				<-handle.listener
				status, _ := handle.task.Status()
				if status > TaskStatus_Processing {
					assert.Equal(t, TaskStatus_Done, status)
					break
				}
			}
		}

		thpWorker.Shutdown()
		thpWorker.Wait()
	})

	t.Run("Task cancel", func(t *testing.T) {
		var ctx = context.Background()
		var thpWorker = NewThreadPoolWorker(ctx, withSchedulingDelay(10*time.Millisecond))
		var handleMap = map[uint64]taskInfo{}

		for i := 0; i < 20; i++ {
			var task = NewTask(func(_ uint64, _ string, _ uint32) {}, withAllNotifier())
			handle, err := thpWorker.Submit(ctx, task)
			if err != nil {
				// t.Logf("Failed to submit task: err = %s", err)
				continue
			}
			ch, _ := handle.Channel()
			handleMap[handle.ID()] = taskInfo{
				task:     handle,
				listener: ch,
			}
			handle.Cancel()
		}

		for _, handle := range handleMap {
			for {
				<-handle.listener
				status, _ := handle.task.Status()
				if status > TaskStatus_Processing {
					assert.Equal(t, TaskStatus_Done, status)
					break
				}
			}
		}

		thpWorker.Shutdown()
		thpWorker.Wait()
	})

	t.Run("Submit timeout", func(t *testing.T) {
		var ctx = context.Background()
		var thpWorker = NewThreadPoolWorker(ctx, WithNoAutoStart())

		for i := 0; i < 20; i++ {
			var task = NewTask(func(_ uint64, _ string, _ uint32) {}, withAllNotifier())
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
		var handleMap = sync.Map{}

		go func() {
			for i := 0; i < 20; i++ {
				var task = NewTask(func(_ uint64, _ string, _ uint32) { panic("test panic") }, withAllNotifier())
				handle, err := thpWorker.Submit(ctx, task)
				if err != nil {
					t.Logf("Failed to submit task: err = %s", err)
					continue
				}
				ch, _ := handle.Channel()
				handleMap.LoadOrStore(handle.ID(), taskInfo{
					task:     handle,
					listener: ch,
				})
			}
		}()

		handleMap.Range(func(key any, value any) bool {
			var handle = value.(taskInfo)
			for {
				<-handle.listener
				status, err := handle.task.Status()
				if status > TaskStatus_Processing {
					assert.Equal(t, TaskStatus_Failed, status, "Task status should be failed")
					assert.NotEqual(t, nil, err, "Panic not observed for: id = %d", handle.task.ID())
					break
				}
			}
			return true
		})

		thpWorker.Shutdown()
		thpWorker.Wait()
	})

	t.Run("Producer", func(t *testing.T) {
		t.Run("Manual start", func(t *testing.T) {
			var ctx = context.Background()
			var thpWorker = NewThreadPoolWorker(ctx, WithNoAutoStart())
			var handleMap = map[uint64]taskInfo{}

			assert.Equal(t, true, thpWorker.producerStop == nil)

			var doneCh = make(chan struct{})

			go func() {
				for i := 0; i < 20; i++ {
					var task = NewTask(func(_ uint64, _ string, _ uint32) {}, withAllNotifier())
					handle, err := thpWorker.Submit(ctx, task)
					if err != nil {
						// t.Logf("Failed to submit task: err = %s", err)
						continue
					}
					ch, _ := handle.Channel()
					handleMap[handle.ID()] = taskInfo{
						task:     handle,
						listener: ch,
					}
				}

				doneCh <- struct{}{}
			}()

			go func() {
				thpWorker.start()
			}()

			<-doneCh

			for _, handle := range handleMap {
				for {
					<-handle.listener
					status, _ := handle.task.Status()
					if status > TaskStatus_Processing {
						status, err := handle.task.Status()
						assert.Equal(t, TaskStatus_Done, status)
						assert.Equal(t, nil, err)
						break
					}
				}
			}

			thpWorker.Shutdown()
			thpWorker.Wait()

		})

		t.Run("Manual stop", func(t *testing.T) {
			var ctx = context.Background()
			var thpWorker = NewThreadPoolWorker(ctx, withSchedulingDelay(10*time.Millisecond))
			var handleMap = map[uint64]taskInfo{}

			assert.Equal(t, false, thpWorker.producerStop == nil)

			var doneCh = make(chan struct{})

			go func() {
				for i := 0; i < 20; i++ {
					var task = NewTask(func(_ uint64, _ string, _ uint32) {}, withAllNotifier())
					handle, err := thpWorker.Submit(ctx, task)
					if err != nil {
						// t.Logf("Failed to submit task: err = %s", err)
						continue
					}
					ch, _ := handle.Channel()
					handleMap[handle.ID()] = taskInfo{
						task:     handle,
						listener: ch,
					}
				}

				doneCh <- struct{}{}
			}()

			var stopCh = make(chan struct{})
			go func() {
				thpWorker.stop()
				stopCh <- struct{}{}
			}()

			<-stopCh
			assert.Equal(t, true, thpWorker.producerStop == nil)

			thpWorker.start()

			<-doneCh

			for _, handle := range handleMap {
				for {
					<-handle.listener
					status, _ := handle.task.Status()
					if status > TaskStatus_Processing {
						status, err := handle.task.Status()
						assert.Equal(t, TaskStatus_Done, status)
						assert.Equal(t, nil, err)
						break
					}
				}
			}

			thpWorker.Shutdown()
			thpWorker.Wait()
		})
	})
}
