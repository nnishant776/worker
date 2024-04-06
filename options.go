package worker

type workerConfig struct {
	queueSize int
	poolSize  int
	autoStart bool
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
