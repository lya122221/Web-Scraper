package engine

import (
	"context"
	"errors"
	"log"
	"sync"
)

var ErrWorkerPoolStopped = errors.New("worker pool stopped")

type WorkerPool struct {
	jobs        chan func()
	workerCount int
	wg          sync.WaitGroup
	stop        chan struct{}
	startOnce   sync.Once
	stateMu     sync.RWMutex
	stopped     bool
}

func NewWorkerPool(workerCount int, jobsCount int) *WorkerPool {
	jobs := make(chan func(), jobsCount)

	return &WorkerPool{
		jobs:        jobs,
		workerCount: workerCount,
		stop:        make(chan struct{}),
	}
}

func (p *WorkerPool) Start(ctx context.Context) {
	p.startOnce.Do(func() {
		p.stateMu.RLock()
		defer p.stateMu.RUnlock()

		if p.stopped {
			return
		}

		for i := 0; i < p.workerCount; i++ {
			p.wg.Add(1)

			go p.worker()
		}

		go func() {
			select {
			case <-ctx.Done():
				p.Stop()
			case <-p.stop:
			}
		}()
	})
}

func (p *WorkerPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.stop:
			log.Println("Worker stopped")
			return
		case job := <-p.jobs:
			p.stateMu.RLock()
			if p.stopped {
				p.stateMu.RUnlock()
				log.Println("Worker stopped")
				return
			}
			p.stateMu.RUnlock()

			job()
		}
	}
}

func (p *WorkerPool) Submit(ctx context.Context, job func()) error {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()

	if p.stopped {
		return ErrWorkerPoolStopped
	}

	select {
	case p.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *WorkerPool) Stop() {
	p.stateMu.Lock()
	if !p.stopped {
		p.stopped = true
		close(p.stop)
	}
	p.stateMu.Unlock()

	p.wg.Wait()
}
