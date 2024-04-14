package worker

type taskConfig struct {
	notifier     chan struct{}
	uuid         string
	highPriority bool
	taskBitmap   [_TASK_STATUS_MAX_]bool
}

type TaskOption func(o taskConfig) taskConfig

func WithUUID(uuid string) TaskOption {
	return func(cfg taskConfig) taskConfig {
		cfg.uuid = uuid
		return cfg
	}
}

func WithNotifier(statuses ...TaskStatus) TaskOption {
	return func(cfg taskConfig) taskConfig {
		if len(statuses) > 0 {
			cfg.notifier = make(chan struct{}, _TASK_STATUS_MAX_)
			for _, status := range statuses {
				cfg.taskBitmap[status] = true
			}
		}
		return cfg
	}
}

func WithHighPriority() TaskOption {
	return func(cfg taskConfig) taskConfig {
		cfg.highPriority = true
		return cfg
	}
}
