package worker

import (
	"context"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var logOutput io.Writer = os.Stdout
var defaultQueueSize = uint32(runtime.GOMAXPROCS(0))
var defaultPoolSize = defaultQueueSize

type ThreadPoolWorker struct {
	orgCtx       context.Context
	ctx          context.Context
	producerStop context.CancelFunc
	shutdown     context.CancelFunc
	taskQueue    chan Task
	runQueue     chan Task
	wg           sync.WaitGroup
	nextID       atomic.Uint64
	cfg          workerConfig
	mu           sync.RWMutex
}

func DefaultPoolSize() uint32 {
	return defaultPoolSize
}

func DefaultQueueSize() uint32 {
	return defaultQueueSize
}

func NewThreadPoolWorker(ctx context.Context, opts ...WorkerOption) *ThreadPoolWorker {
	var workerCfg = workerConfig{
		autoStart:   true,
		autoRespawn: true,
		queueSize:   defaultQueueSize,
		poolSize:    defaultPoolSize,
	}

	for _, opt := range opts {
		workerCfg = opt(workerCfg)
	}

	var thpWorker = &ThreadPoolWorker{
		orgCtx:    ctx,
		taskQueue: make(chan Task, workerCfg.queueSize),
		runQueue:  make(chan Task, workerCfg.poolSize),
		cfg:       workerCfg,
	}

	thpWorker.startWorkers(workerCfg.poolSize)

	if workerCfg.autoStart {
		thpWorker.startProducer()
	}

	return thpWorker
}

func (self *ThreadPoolWorker) Submit(ctx context.Context, task Task) (*TaskHandle, error) {
	var th = &TaskHandle{
		id:       self.nextID.Add(1),
		status:   TaskStatus_Submitted,
		uuid:     task.cfg.uuid,
		notifier: task.cfg.notifier,
	}

	var ch chan Task
	if task.cfg.highPriority {
		ch = self.runQueue
		th.updateStatus(TaskStatus_Queued, nil, task.cfg.taskBitmap)
	} else {
		ch = self.taskQueue
		th.updateStatus(TaskStatus_Submitted, nil, task.cfg.taskBitmap)
	}

	task.handle = th

	select {
	case ch <- task:
	case <-ctx.Done():
		return th, ctx.Err()
	}

	return th, nil
}

func (self *ThreadPoolWorker) start() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.startProducer()
}

func (self *ThreadPoolWorker) stop() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.stopProducer()
}

func (self *ThreadPoolWorker) Wait() {
	self.wait()
}

func (self *ThreadPoolWorker) Shutdown() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.stopWorkers()
}

func (self *ThreadPoolWorker) wait() {
	self.wg.Wait()
}

func (self *ThreadPoolWorker) runWorker(ctx context.Context, id uint32, wg *sync.WaitGroup) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()

	var task Task

	defer func() {
		if err := recover(); err != nil {
			task.handle.updateStatus(TaskStatus_Failed, err, task.cfg.taskBitmap)
			if self.cfg.autoRespawn {
				wg = nil
				go self.runWorker(ctx, id, &self.wg)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case task = <-self.runQueue:
			status, _ := task.handle.Status()
			if status >= TaskStatus_Cancelled {
				continue
			}

			task.handle.updateStatus(TaskStatus_Processing, nil, task.cfg.taskBitmap)
			task.Run(task.handle.id, task.handle.uuid, id, task.cfg.taskArgs...)
			task.handle.updateStatus(TaskStatus_Done, nil, task.cfg.taskBitmap)
		}
	}
}

func (self *ThreadPoolWorker) runProducer(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case task := <-self.taskQueue:
			if self.cfg.schedDelay > 0 {
				time.Sleep(self.cfg.schedDelay)
			}
			task.handle.updateStatus(TaskStatus_Queued, nil, task.cfg.taskBitmap)
			self.runQueue <- task
		}
	}
}

func (self *ThreadPoolWorker) startProducer() {
	self.stopProducer()

	producerCtx, producerCancel := context.WithCancel(self.ctx)
	self.producerStop = producerCancel

	self.wg.Add(1)
	go self.runProducer(producerCtx, &self.wg)
}

func (self *ThreadPoolWorker) stopProducer() {
	if self.producerStop != nil {
		self.producerStop()
		self.producerStop = nil
	}
}

func (self *ThreadPoolWorker) stopWorkers() {
	if self.shutdown != nil {
		self.shutdown()
		self.shutdown = nil
	}
}

func (self *ThreadPoolWorker) startWorkers(count uint32) {
	self.stopWorkers()

	workerCtx, workerCancel := context.WithCancel(self.orgCtx)
	self.ctx = workerCtx
	self.shutdown = workerCancel

	for i := uint32(0); i < count; i++ {
		self.wg.Add(1)
		go self.runWorker(workerCtx, i, &self.wg)
	}
}
