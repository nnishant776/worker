package worker

import "context"

type Worker interface {
	Submit(context.Context, Task) (*TaskHandle, error)
	Shutdown()
	Wait()
}

var _ Worker = (*ThreadPoolWorker)(nil)
