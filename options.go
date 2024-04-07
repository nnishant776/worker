package worker

import "time"

type workerConfig struct {
	queueSize   int
	poolSize    int
	autoStart   bool
	autoRespawn bool
	schedDelay  time.Duration
}

type workerOptFunc func(*workerConfig)

func (self workerOptFunc) apply(cfg *workerConfig) {
	self(cfg)
}

type Option interface {
	apply(*workerConfig)
}

func WithQueueSize(size int) Option {
	return workerOptFunc(func(cfg *workerConfig) {
		cfg.queueSize = size
	})
}

func WithPoolSize(size int) Option {
	return workerOptFunc(func(cfg *workerConfig) {
		cfg.poolSize = size
	})
}

func WithNoAutoStart() Option {
	return workerOptFunc(func(cfg *workerConfig) {
		cfg.autoStart = false
	})
}

func WithNoWorkerAutoRespawn() Option {
	return workerOptFunc(func(cfg *workerConfig) {
		cfg.autoRespawn = false
	})
}
