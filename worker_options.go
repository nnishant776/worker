package worker

import "time"

type workerConfig struct {
	queueSize   uint32
	poolSize    uint32
	autoStart   bool
	autoRespawn bool
	schedDelay  time.Duration
}

type WorkerOption func(cfg workerConfig) workerConfig

func WithQueueSize(size uint32) WorkerOption {
	return func(cfg workerConfig) workerConfig {
		cfg.queueSize = size
		return cfg
	}
}

func WithPoolSize(size uint32) WorkerOption {
	return func(cfg workerConfig) workerConfig {
		cfg.poolSize = size
		return cfg
	}
}

func WithNoAutoStart() WorkerOption {
	return func(cfg workerConfig) workerConfig {
		cfg.autoStart = false
		return cfg
	}
}

func WithNoWorkerAutoRespawn() WorkerOption {
	return func(cfg workerConfig) workerConfig {
		cfg.autoRespawn = false
		return cfg
	}
}
