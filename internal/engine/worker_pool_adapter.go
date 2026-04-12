package engine

import (
	"context"

	"neo_collector_go/internal/scheduler"
)

type WorkerPoolAdapter struct {
	workers int
	pool    *scheduler.WorkerPool
}

func NewWorkerPoolAdapter(pool *scheduler.WorkerPool, workers int) *WorkerPoolAdapter {
	if workers <= 0 {
		workers = 1
	}

	return &WorkerPoolAdapter{
		workers: workers,
		pool:    pool,
	}
}

func (a *WorkerPoolAdapter) Run(ctx context.Context, total int, fn func(context.Context, int) error) error {
	if a == nil || a.pool == nil {
		return scheduler.NewWorkerPool(1).Run(ctx, total, fn)
	}
	return a.pool.Run(ctx, total, fn)
}
