package worker

import "context"

type Worker interface {
	Submit(context.Context, Task) (TaskHandle, error)
}

var _ Worker = (*ThreadPoolWorker)(nil)