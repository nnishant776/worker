package worker

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

type ThreadPoolWorker struct {
	orgCtx    context.Context
	ctx       context.Context
	stop      context.CancelFunc
	shutdown  context.CancelFunc
	taskQueue chan Task
	runQueue  chan Task
	wg        sync.WaitGroup
	nextID    atomic.Uint64
	cfg       workerConfig
	mu        sync.RWMutex
}

func NewThreadPoolWorker(ctx context.Context, opts ...Option) *ThreadPoolWorker {
	var workerCfg = workerConfig{
		autoStart: true,
		queueSize: runtime.GOMAXPROCS(0),
		poolSize:  runtime.GOMAXPROCS(0),
	}

	for _, opt := range opts {
		opt.apply(&workerCfg)
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

func (self *ThreadPoolWorker) Submit(ctx context.Context, task Task) (TaskHandle, error) {
	var th = TaskHandle{
		id:     self.nextID.Add(1),
		done:   make(chan struct{}),
		cancel: make(chan struct{}),
		panic:  make(chan any, 1),
	}
	task.handle = th
	var ch chan Task
	if task.isHighPriority {
		fmt.Printf("Sending task in run queue: id = %d\n", task.handle.id)
		ch = self.runQueue
	} else {
		fmt.Printf("Sending task in task queue: id = %d\n", task.handle.id)
		ch = self.taskQueue
	}

	select {
	case ch <- task:
	case <-ctx.Done():
		return th, ctx.Err()
	}

	return th, nil
}

func (self *ThreadPoolWorker) Start() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.startProducer()
}

func (self *ThreadPoolWorker) Stop() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.stopProducer()
}

func (self *ThreadPoolWorker) Wait() {
	self.mu.Lock()
	defer self.mu.Unlock()
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

func (self *ThreadPoolWorker) runWorker(ctx context.Context, id int, wg *sync.WaitGroup) {
	fmt.Printf("Started worker: %d\n", id)
	if wg != nil {
		defer func() {
			wg.Done()
			fmt.Printf("Decremented the wg counter\n")
			fmt.Printf("Stopped worker thread: id = %d\n", id)
		}()
	}

	var task Task

	defer func() {
		err := recover()
		if err != nil {
			fmt.Printf("Thread paniced: %+v\n", err)
			task.handle.panic <- err
			fmt.Printf("Worker thread paniced. Respawning: id = %d\n", id)
			wg.Add(1)
			go self.runWorker(ctx, id, wg)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case task = <-self.runQueue:
			if task.handle.IsCancelled() {
				fmt.Printf("Task is already cancelled. Ignoring: id = %d\n", task.handle.id)
				continue
			}
			fmt.Printf("Handling task: id = %d\n", task.handle.id)
			task.Run()
		}
	}
}

func (self *ThreadPoolWorker) runProducer(ctx context.Context, wg *sync.WaitGroup) {
	fmt.Printf("Started producer thread\n")
	if wg != nil {
		defer func() {
			wg.Done()
			fmt.Printf("Stopped producer thread\n")
		}()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case task := <-self.taskQueue:
			self.runQueue <- task
		}
	}
}

func (self *ThreadPoolWorker) startProducer() {
	self.stopProducer()

	producerCtx, producerCancel := context.WithCancel(self.ctx)
	self.stop = producerCancel

	self.wg.Add(1)
	go self.runProducer(producerCtx, &self.wg)
}

func (self *ThreadPoolWorker) stopProducer() {
	if self.stop != nil {
		fmt.Printf("Stopping producer thread\n")
		self.stop()
		self.stop = nil
	}
}

func (self *ThreadPoolWorker) stopWorkers() {
	if self.shutdown != nil {
		fmt.Printf("Stopping worker threads\n")
		self.shutdown()
		self.shutdown = nil
	}
}

func (self *ThreadPoolWorker) startWorkers(count int) {
	self.stopWorkers()

	workerCtx, workerCancel := context.WithCancel(self.orgCtx)
	self.ctx = workerCtx
	self.shutdown = workerCancel

	for i := 0; i < count; i++ {
		self.wg.Add(1)
		go self.runWorker(workerCtx, i, &self.wg)
	}
}
