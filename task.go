package worker

type Task struct {
	Run            func()
	handle         TaskHandle
	isHighPriority bool
}

func (self *Task) SetHighPriority() {
	self.isHighPriority = true
}

type TaskHandle struct {
	done   chan struct{}
	cancel chan struct{}
	panic  chan any
	id     uint64
}

func (self TaskHandle) ID() uint64 {
	return self.id
}

func (self TaskHandle) IsDone() bool {
	select {
	case <-self.done:
		return true
	default:
		return false
	}
}

func (self TaskHandle) Done() <-chan struct{} {
	return self.done
}

func (self TaskHandle) IsCancelled() bool {
	select {
	case <-self.cancel:
		return true
	default:
		return false
	}
}

func (self TaskHandle) Cancel() {
	select {
	case <-self.cancel:
	default:
		close(self.cancel)
	}
}

func (self TaskHandle) Panic() any {
	select {
	case err := <-self.panic:
		return err
	default:
		return nil
	}
}