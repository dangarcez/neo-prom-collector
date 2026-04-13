package neo4j

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	driver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"neo_collector_go/internal/config"
	"neo_collector_go/internal/domain"
)

type Repository struct {
	driver            driver.DriverWithContext
	database          string
	logger            *slog.Logger
	nodeLocks         sync.Map
	relationshipLocks sync.Map
}

func NewRepository(ctx context.Context, env config.EnvConfig, logger *slog.Logger) (*Repository, error) {
	if strings.TrimSpace(env.Neo4jPassword) == "" {
		return nil, fmt.Errorf("NEO4J_PASSWORD is required when dry_run is false")
	}

	repoDriver, err := driver.NewDriverWithContext(
		env.Neo4jURI,
		driver.BasicAuth(env.Neo4jUsername, env.Neo4jPassword, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("create neo4j driver: %w", err)
	}

	if env.VerifyConnectivity {
		verifyCtx, cancel := context.WithTimeout(ctx, env.Neo4jTimeout)
		defer cancel()

		if err := repoDriver.VerifyConnectivity(verifyCtx); err != nil {
			_ = repoDriver.Close(ctx)
			return nil, fmt.Errorf("verify neo4j connectivity: %w", err)
		}
	}

	return &Repository{
		driver:   repoDriver,
		database: env.Neo4jDatabase,
		logger:   logger,
	}, nil
}

func (r *Repository) Close(ctx context.Context) error {
	if r == nil || r.driver == nil {
		return nil
	}

	return r.driver.Close(ctx)
}

func (r *Repository) ApplyPlan(ctx context.Context, plan domain.MutationPlan) (domain.ApplyStats, error) {
	if r == nil || r.driver == nil {
		return domain.ApplyStats{}, fmt.Errorf("neo4j repository is not initialized")
	}

	session := r.driver.NewSession(ctx, driver.SessionConfig{
		DatabaseName: r.database,
	})
	defer session.Close(ctx)

	rawResult, err := session.ExecuteWrite(ctx, func(tx driver.ManagedTransaction) (any, error) {
		stats := domain.ApplyStats{}

		for _, node := range plan.Nodes {
			action, err := r.applyNode(ctx, tx, node)
			if err != nil {
				return stats, err
			}
			stats.AddNode(action)
		}

		for _, relationship := range plan.Relationships {
			relationshipStats, err := r.applyRelationship(ctx, tx, relationship)
			if err != nil {
				return stats, err
			}
			stats.Merge(relationshipStats)
		}

		return stats, nil
	})
	if err != nil {
		return domain.ApplyStats{}, err
	}

	stats, ok := rawResult.(domain.ApplyStats)
	if !ok {
		return domain.ApplyStats{}, fmt.Errorf("unexpected transaction result type %T", rawResult)
	}

	return stats, nil
}
