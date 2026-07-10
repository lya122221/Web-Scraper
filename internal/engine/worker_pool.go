package engine

import (
	"context"
	"log"
	"sync"
)

type WorkerPool struct {
	jobs        chan func()
	workerCount int
	wg          sync.WaitGroup
}

func NewWorker(workerCount int, jobsCount int) *WorkerPool {

	jobs := make(chan func(), jobsCount)

	return &WorkerPool{
		jobs,
		workerCount,
		sync.WaitGroup{},
	}
}

func (p *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)

		go p.worker(ctx)
	}
}

func (p *WorkerPool) worker(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Println("context canceled")
			return
		case job, ok := <-p.jobs:
			if !ok {
				log.Println("jobs chan closed")
				return
			}

			job()
		}
	}
}

func (p *WorkerPool) Submit(job func()) {
	p.jobs <- job
}

func (p *WorkerPool) Stop() {
	close(p.jobs)
	p.wg.Wait()
}
