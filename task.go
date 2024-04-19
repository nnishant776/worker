package worker

import (
	"errors"
	"sync"
)

type TaskStatus int32

const (
	TaskStatus_Unknown TaskStatus = iota
	TaskStatus_Submitted
	TaskStatus_Queued
	TaskStatus_Cancelled
	TaskStatus_Processing
	TaskStatus_Done
	TaskStatus_Failed
	_TASK_STATUS_MAX_
)

var ErrNoSubscriber = errors.New("no listener for this task")

type TaskFunc = func(taskID uint64, uuid string, workerID uint32, args ...any)

type Task struct {
	Run    TaskFunc
	handle *TaskHandle
	cfg    taskConfig
}

func NewTask(job TaskFunc, opts ...TaskOption) Task {
	var taskCfg taskConfig

	for _, opt := range opts {
		taskCfg = opt(taskCfg)
	}

	return Task{
		Run: job,
		cfg: taskCfg,
	}
}

type TaskHandle struct {
	panicData any
	notifier  chan struct{}
	uuid      string
	id        uint64
	mu        sync.RWMutex
	status    TaskStatus
}

func (self *TaskHandle) ID() uint64 {
	return self.id
}

func (self *TaskHandle) UUID() string {
	return self.uuid
}

func (self *TaskHandle) Cancel() {
	self.mu.Lock()
	defer self.mu.Unlock()
	if self.status < TaskStatus_Cancelled {
		self.status = TaskStatus_Cancelled
		if self.notifier != nil {
			select {
			case self.notifier <- struct{}{}:
			default:
			}
		}
	}
}

func (self *TaskHandle) Status() (TaskStatus, any) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.status, self.panicData
}

func (self *TaskHandle) Channel() (<-chan struct{}, error) {
	if self.notifier == nil {
		return nil, ErrNoSubscriber
	}

	return self.notifier, nil
}

func (self *TaskHandle) updateStatus(status TaskStatus, panicData any, bitMap [_TASK_STATUS_MAX_]bool) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.status = status
	if panicData != nil {
		self.panicData = panicData
	}
	if bitMap[status] && self.notifier != nil {
		select {
		case self.notifier <- struct{}{}:
		default:
		}
	}
}
