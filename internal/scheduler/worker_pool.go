package scheduler

import (
	"context"
	"errors"
	"sync"
)

type WorkerPool struct {
	workers int
}

func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = 1
	}

	return &WorkerPool{workers: workers}
}

func (p *WorkerPool) Run(ctx context.Context, total int, fn func(context.Context, int) error) error {
	if total <= 0 {
		return nil
	}

	if p == nil || p.workers <= 1 || total == 1 {
		var joined error
		for index := 0; index < total; index++ {
			if err := fn(ctx, index); err != nil {
				joined = errors.Join(joined, err)
			}
		}
		return joined
	}

	workerCount := p.workers
	if total < workerCount {
		workerCount = total
	}

	indexes := make(chan int)
	errs := make(chan error, total)

	var wg sync.WaitGroup
	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for index := range indexes {
				if err := fn(ctx, index); err != nil {
					errs <- err
				}
			}
		}()
	}

	for index := 0; index < total; index++ {
		select {
		case <-ctx.Done():
			close(indexes)
			wg.Wait()
			close(errs)
			var joined error
			for err := range errs {
				joined = errors.Join(joined, err)
			}
			return errors.Join(ctx.Err(), joined)
		case indexes <- index:
		}
	}

	close(indexes)
	wg.Wait()
	close(errs)

	var joined error
	for err := range errs {
		joined = errors.Join(joined, err)
	}

	return joined
}
