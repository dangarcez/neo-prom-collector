package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"neo_collector_go/internal/observability"
)

type job struct {
	targetName string
	jobName    string
	interval   time.Duration
	run        func(context.Context) error
	running    atomic.Bool
}

type Scheduler struct {
	logger  *slog.Logger
	metrics *observability.Metrics
	once    bool
	jobs    []*job
}

func New(logger *slog.Logger, metrics *observability.Metrics, once bool) *Scheduler {
	return &Scheduler{
		logger:  logger,
		metrics: metrics,
		once:    once,
		jobs:    []*job{},
	}
}

func (s *Scheduler) AddJob(targetName, jobName string, interval time.Duration, run func(context.Context) error) {
	s.jobs = append(s.jobs, &job{
		targetName: targetName,
		jobName:    jobName,
		interval:   interval,
		run:        run,
	})
}

func (s *Scheduler) Run(ctx context.Context) error {
	if s.once {
		var joined error
		for _, job := range s.jobs {
			joined = errors.Join(joined, s.execute(ctx, job))
		}
		return joined
	}

	var wg sync.WaitGroup
	wg.Add(len(s.jobs))

	for _, scheduledJob := range s.jobs {
		go func(currentJob *job) {
			defer wg.Done()
			s.loop(ctx, currentJob)
		}(scheduledJob)
	}

	<-ctx.Done()
	wg.Wait()
	return nil
}

func (s *Scheduler) loop(ctx context.Context, job *job) {
	_ = s.execute(ctx, job)

	ticker := time.NewTicker(job.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if job.running.Load() {
				s.metrics.RecordJobSkipped()
				s.logger.Warn("job execution skipped because the previous run is still active",
					"target", job.targetName,
					"job", job.jobName,
				)
				continue
			}

			_ = s.execute(ctx, job)
		}
	}
}

func (s *Scheduler) execute(ctx context.Context, job *job) error {
	if !job.running.CompareAndSwap(false, true) {
		return nil
	}
	defer job.running.Store(false)

	s.metrics.RecordJobStarted()
	startedAt := time.Now()
	err := job.run(ctx)
	s.metrics.RecordJobFinished(err)

	if err != nil {
		s.logger.Error("job execution failed",
			"target", job.targetName,
			"job", job.jobName,
			"duration", time.Since(startedAt).String(),
			"error", err.Error(),
		)
		return fmt.Errorf("target %s job %s: %w", job.targetName, job.jobName, err)
	}

	s.logger.Info("job execution finished",
		"target", job.targetName,
		"job", job.jobName,
		"duration", time.Since(startedAt).String(),
	)

	return nil
}
