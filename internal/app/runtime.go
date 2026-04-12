package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"neo_collector_go/internal/observability"
	"neo_collector_go/internal/scheduler"
)

type repositoryCloser interface {
	Close(ctx context.Context) error
}

type Runtime struct {
	Logger     *slog.Logger
	Metrics    *observability.Metrics
	Scheduler  *scheduler.Scheduler
	Repository repositoryCloser
}

func (r *Runtime) Run(ctx context.Context) error {
	runErr := r.Scheduler.Run(ctx)
	r.Metrics.LogSnapshot(r.Logger)

	if r.Repository == nil {
		return runErr
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return errors.Join(runErr, r.Repository.Close(closeCtx))
}
