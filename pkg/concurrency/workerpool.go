package concurrency

import (
	"errors"
	"sync"
)

var ErrPoolStopped = errors.New("concurrency: pool is stopped")

type Pool struct {
	workers  int
	tasks    chan func()
	wg       sync.WaitGroup
	mu       sync.RWMutex
	stopped  bool
	started  bool
	stopOnce sync.Once
}

func NewPool(workers, queueSize int) *Pool {
	if workers < 1 {
		workers = 1
	}
	if queueSize < 0 {
		queueSize = 0
	}
	return &Pool{
		workers: workers,
		tasks:   make(chan func(), queueSize),
	}
}

func (p *Pool) Start() {
	p.mu.Lock()
	if p.started || p.stopped {
		p.mu.Unlock()
		return
	}
	p.started = true
	p.mu.Unlock()

	p.wg.Add(p.workers)
	for i := 0; i < p.workers; i++ {
		go p.worker()
	}
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for task := range p.tasks {
		task()
	}
}

func (p *Pool) Submit(task func()) error {
	if task == nil {
		return nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.stopped {
		return ErrPoolStopped
	}
	p.tasks <- task
	return nil
}

func (p *Pool) TrySubmit(task func()) bool {
	if task == nil {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.stopped {
		return false
	}
	select {
	case p.tasks <- task:
		return true
	default:
		return false
	}
}

func (p *Pool) Stop() {
	p.stopOnce.Do(func() {
		p.mu.Lock()
		p.stopped = true
		started := p.started
		close(p.tasks)
		p.mu.Unlock()

		if started {
			p.wg.Wait()
		}
	})
}

func (p *Pool) QueueLen() int {
	return len(p.tasks)
}
