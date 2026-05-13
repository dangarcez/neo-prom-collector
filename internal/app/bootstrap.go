package app

import (
	"context"
	"fmt"
	"strings"

	"neo_collector_go/internal/collector/prometheus"
	"neo_collector_go/internal/config"
	"neo_collector_go/internal/engine"
	"neo_collector_go/internal/observability"
	neo4jrepo "neo_collector_go/internal/repository/neo4j"
	"neo_collector_go/internal/scheduler"
)

type Options struct {
	EnvPath    string
	ConfigPath string
	Once       bool
}

func Bootstrap(ctx context.Context, opts Options) (*Runtime, error) {
	envConfig, err := config.LoadEnv(opts.EnvPath)
	if err != nil {
		return nil, fmt.Errorf("load env config: %w", err)
	}

	configPath := strings.TrimSpace(opts.ConfigPath)
	if configPath == "" {
		configPath = envConfig.ConfigPath
	}

	fileConfig, err := config.LoadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("load file config: %w", err)
	}

	logger := observability.NewLogger(envConfig.LogLevel, envConfig.LogFormat)
	metrics := observability.NewMetrics()
	logger.Info("configuration loaded", "config_path", configPath, "targets", len(fileConfig.PromTargets))

	var repository *neo4jrepo.Repository
	if !fileConfig.AllTargetsDryRun() {
		repository, err = neo4jrepo.NewRepository(ctx, envConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("create neo4j repository: %w", err)
		}
	}

	planner := engine.NewPlanner()
	workerPool := scheduler.NewWorkerPool(envConfig.MaxDatapointWorkers)
	processor := engine.NewProcessor(
		logger,
		planner,
		repository,
		engine.NewWorkerPoolAdapter(workerPool, envConfig.MaxDatapointWorkers),
		metrics,
	)
	jobScheduler := scheduler.New(logger, metrics, opts.Once)

	for _, target := range fileConfig.PromTargets {
		target := target
		client, err := prometheus.NewClient(target)
		if err != nil {
			return nil, fmt.Errorf("create prometheus client for target %q: %w", target.Name, err)
		}

		for _, job := range target.Jobs {
			job := job
			jobScheduler.AddJob(target.Name, job.Name, job.Interval(), func(runCtx context.Context) error {
				return processor.ProcessJob(runCtx, target, job, client)
			})
		}
	}

	return &Runtime{
		Logger:     logger,
		Metrics:    metrics,
		Scheduler:  jobScheduler,
		Repository: repository,
	}, nil
}
