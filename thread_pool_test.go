package worker_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/nnishant776/worker"
	"github.com/stretchr/testify/assert"
)

func Test_ThreadPoolWorker(t *testing.T) {
	t.Logf("Default queue size: %v", runtime.GOMAXPROCS(0))

	var ctx = context.Background()
	var thpWorker = worker.NewThreadPoolWorker(ctx, worker.WithNoAutoStart())
	var handleMap = map[int]worker.TaskHandle{}

	for i := 0; i < 20; i++ {
		var task = worker.Task{
			Run: func() {
				panic("test panic")
			},
		}
		// task.SetHighPriority()
		chCtx, _ := context.WithTimeout(ctx, 5*time.Second)
		handle, err := thpWorker.Submit(chCtx, task)
		if err != nil {
			t.Logf("Failed to submit task: err = %s", err)
			continue
		}
		if i%2 == 0 {
			handle.Cancel()
		}
		handleMap[i] = handle
	}

	time.Sleep(5 * time.Second)
	t.Logf("Finished sleep")
	thpWorker.Shutdown()
	thpWorker.Wait()

	for k, handle := range handleMap {
		if k%2 == 0 {
			assert.Equal(t, true, handle.IsCancelled())
		} else {
			assert.NotEqual(t, nil, handle.Panic())
		}
	}
}
