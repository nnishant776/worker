package worker

import "time"

type workerConfig struct {
	queueSize   uint32
	poolSize    uint32
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

func WithQueueSize(size uint32) Option {
	return workerOptFunc(func(cfg *workerConfig) {
		cfg.queueSize = size
	})
}

func WithPoolSize(size uint32) Option {
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
