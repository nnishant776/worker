package benchmark

import (
	"context"
	"testing"

	"github.com/nnishant776/worker"
)

func Benchmark_ThreadPoolWorker(b *testing.B) {
	b.ReportAllocs()
	b.Run("Default configuration", func(b *testing.B) {
		b.Run("1000 jobs", func(b *testing.B) {
			var ctx = context.Background()
			var thpWorker = worker.NewThreadPoolWorker(ctx)
			type taskInfo struct {
				task     *worker.TaskHandle
				listener chan struct{}
			}
			var taskList = make([]taskInfo, 0, 1000)

			for i := 0; i < b.N; i++ {
				taskList = taskList[:0]
				for j := 0; j < 1000; j++ {
					handle, err := thpWorker.Submit(ctx, worker.NewTask(func() {}))

					taskList = append(taskList, taskInfo{
						task: handle,
					})

					if err != nil {
						b.Errorf("Failed to submit task: err = %s", err)
						b.FailNow()
					}
				}
			}

			thpWorker.Shutdown()
			thpWorker.Wait()
		})
	})
}
