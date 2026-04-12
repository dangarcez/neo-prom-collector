package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"neo_collector_go/internal/config"
	"neo_collector_go/internal/domain"
	"neo_collector_go/internal/observability"
)

type Collector interface {
	Query(ctx context.Context, query string) ([]domain.Datapoint, error)
}

type Repository interface {
	ApplyPlan(ctx context.Context, plan domain.MutationPlan) (domain.ApplyStats, error)
}

type WorkerPool interface {
	Run(ctx context.Context, total int, fn func(context.Context, int) error) error
}

type Processor struct {
	logger     *slog.Logger
	planner    *Planner
	repository Repository
	workerPool WorkerPool
	metrics    *observability.Metrics
}

func NewProcessor(
	logger *slog.Logger,
	planner *Planner,
	repository Repository,
	workerPool WorkerPool,
	metrics *observability.Metrics,
) *Processor {
	return &Processor{
		logger:     logger,
		planner:    planner,
		repository: repository,
		workerPool: workerPool,
		metrics:    metrics,
	}
}

func (p *Processor) ProcessJob(ctx context.Context, target config.PromTargetConfig, job config.JobConfig, collector Collector) error {
	datapoints, err := collector.Query(ctx, job.Query)
	if err != nil {
		return fmt.Errorf("query prometheus: %w", err)
	}

	stats := domain.ProcessStats{
		Datapoints: len(datapoints),
	}

	workers := 1
	pool := p.workerPool
	if pool == nil {
		pool = NewWorkerPoolAdapter(nil, 1)
	}

	if pool, ok := pool.(*WorkerPoolAdapter); ok {
		workers = pool.workers
	}

	var joined error
	var mu sync.Mutex

	processErr := pool.Run(ctx, len(datapoints), func(runCtx context.Context, index int) error {
		applyStats, err := p.processDatapoint(runCtx, target, job, datapoints[index], index)

		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			stats.Errors++
			joined = errors.Join(joined, err)
		} else {
			stats.ApplyStats.Merge(applyStats)
		}

		return nil
	})
	if processErr != nil {
		joined = errors.Join(joined, processErr)
	}

	p.metrics.RecordProcessStats(stats)
	p.logger.Info("job processing summary",
		"target", target.Name,
		"job", job.Name,
		"datapoints", stats.Datapoints,
		"errors", stats.Errors,
		"nodes_created", stats.NodesCreated,
		"nodes_updated", stats.NodesUpdated,
		"nodes_skipped", stats.NodesSkipped,
		"relationships_created", stats.RelationshipsCreated,
		"relationships_updated", stats.RelationshipsUpdated,
		"relationships_skipped", stats.RelationshipsSkipped,
		"workers", workers,
		"dry_run", target.Runtime.DryRun,
	)

	return joined
}

func (p *Processor) processDatapoint(
	ctx context.Context,
	target config.PromTargetConfig,
	job config.JobConfig,
	datapoint domain.Datapoint,
	index int,
) (domain.ApplyStats, error) {
	plan, err := p.planner.Plan(job, datapoint)
	if err != nil {
		return domain.ApplyStats{}, fmt.Errorf("plan datapoint %d: %w", index, err)
	}

	if target.Runtime.DryRun {
		stats := domain.ApplyStats{}
		stats.NodesSkipped += len(plan.Nodes)
		stats.RelationshipsSkipped += len(plan.Relationships)
		if err := sleepIfNeeded(ctx, target.Runtime.SleepSeconds); err != nil {
			return stats, err
		}
		return stats, nil
	}

	if p.repository == nil {
		return domain.ApplyStats{}, fmt.Errorf("neo4j repository is not configured")
	}

	applyStats, err := p.repository.ApplyPlan(ctx, plan)
	if err != nil {
		return domain.ApplyStats{}, fmt.Errorf("apply datapoint %d: %w", index, err)
	}

	if err := sleepIfNeeded(ctx, target.Runtime.SleepSeconds); err != nil {
		return applyStats, err
	}

	return applyStats, nil
}

func sleepIfNeeded(ctx context.Context, seconds int) error {
	if seconds <= 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(seconds) * time.Second):
		return nil
	}
}
