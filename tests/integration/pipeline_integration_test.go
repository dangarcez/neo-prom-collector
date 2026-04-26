//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	driver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"neo_collector_go/internal/collector/prometheus"
	"neo_collector_go/internal/config"
	"neo_collector_go/internal/domain"
	"neo_collector_go/internal/engine"
	"neo_collector_go/internal/observability"
	neo4jrepo "neo_collector_go/internal/repository/neo4j"
	"neo_collector_go/internal/scheduler"
)

const (
	defaultPrometheusImage = "prom/prometheus:v2.49.1"
	defaultNeo4jImage      = "neo4j:5-community"
	neo4jPassword          = "integration-password"
)

func TestPrometheusCollectorAgainstRealPrometheus(t *testing.T) {
	tc.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	promContainer, promURL := startPrometheusContainer(ctx, t)
	defer terminateContainer(context.Background(), t, promContainer)

	client := prometheus.NewClient(config.PromTargetConfig{
		Name:           "integration-prometheus",
		BaseURL:        promURL,
		TimeoutSeconds: 5,
	})

	datapoints := waitForPrometheusDatapoints(ctx, t, client, "prometheus_build_info")
	if len(datapoints) == 0 {
		t.Fatal("expected prometheus to return at least one datapoint")
	}
	if datapoints[0].Labels["version"] == "" {
		t.Fatalf("expected prometheus_build_info to expose version label, got: %#v", datapoints[0].Labels)
	}
}

func TestProcessorPipelineWithRealPrometheusAndNeo4j(t *testing.T) {
	tc.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	promContainer, promURL := startPrometheusContainer(ctx, t)
	defer terminateContainer(context.Background(), t, promContainer)

	neoContainer, neo4jURI := startNeo4jContainer(ctx, t)
	defer terminateContainer(context.Background(), t, neoContainer)

	promClient := prometheus.NewClient(config.PromTargetConfig{
		Name:           "main_prometheus",
		BaseURL:        promURL,
		TimeoutSeconds: 5,
		Runtime: config.TargetRuntimeConfig{
			DryRun: false,
		},
	})
	datapoints := waitForPrometheusDatapoints(ctx, t, promClient, "prometheus_build_info")
	if len(datapoints) == 0 {
		t.Fatal("expected prometheus to return datapoints before processing")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	metrics := observability.NewMetrics()

	repository := waitForNeo4jRepository(ctx, t, config.EnvConfig{
		Neo4jURI:           neo4jURI,
		Neo4jDatabase:      "neo4j",
		Neo4jUsername:      "neo4j",
		Neo4jPassword:      neo4jPassword,
		Neo4jTimeout:       5 * time.Second,
		VerifyConnectivity: true,
	}, logger)
	defer func() {
		if err := repository.Close(context.Background()); err != nil {
			t.Fatalf("close repository: %v", err)
		}
	}()

	processor := engine.NewProcessor(
		logger,
		engine.NewPlanner(),
		repository,
		engine.NewWorkerPoolAdapter(scheduler.NewWorkerPool(2), 2),
		metrics,
	)

	target := config.PromTargetConfig{
		Name:           "main_prometheus",
		BaseURL:        promURL,
		TimeoutSeconds: 5,
		Runtime: config.TargetRuntimeConfig{
			DryRun: false,
		},
	}
	job := config.JobConfig{
		Name:            "prometheus_build",
		Query:           "prometheus_build_info",
		IntervalSeconds: 1,
		Nodes: []config.NodeTemplateConfig{
			{
				Types:          []string{"Prometheus"},
				TemplateHashes: []string{"prometheus-instance-v1"},
				UpdatePolicy:   "merge",
				StaticProperties: map[string]any{
					"name": "main_prometheus",
					"kind": "prometheus",
				},
				LabelProperties: map[string]string{
					"version": "version",
					"branch":  "branch",
				},
			},
			{
				Types:          []string{"BuildVersion"},
				TemplateHashes: []string{"prometheus-build-v1"},
				UpdatePolicy:   "merge",
				LabelProperties: map[string]string{
					"name":     "version",
					"revision": "revision",
					"go":       "goversion",
				},
			},
		},
		Relationships: []config.RelationshipTemplateConfig{
			{
				Type:         "EXPOSES_BUILD",
				TemplateHash: "prometheus-exposes-build-v1",
				UpdatePolicy: "merge",
				Source: config.RelationshipEndpointConfig{
					Type: "Prometheus",
					MatchAttributes: config.SelectorAttributes{
						Static: map[string]any{
							"name": "main_prometheus",
						},
					},
				},
				Target: config.RelationshipEndpointConfig{
					Type: "BuildVersion",
					MatchAttributes: config.SelectorAttributes{
						Labels: map[string]string{
							"name": "version",
						},
					},
				},
				StaticProperties: map[string]any{
					"source_system": "prometheus",
				},
			},
		},
	}
	job.Normalize(1)

	if err := processor.ProcessJob(ctx, target, job, promClient); err != nil {
		t.Fatalf("first processing run failed: %v", err)
	}
	if err := processor.ProcessJob(ctx, target, job, promClient); err != nil {
		t.Fatalf("second processing run failed: %v", err)
	}

	verifyGraphState(ctx, t, neo4jURI)

	snapshot := metrics.Snapshot()
	if snapshot.JobsStarted != 0 {
		t.Fatalf("processor metrics should not record scheduler job counters, got: %#v", snapshot)
	}
	if snapshot.NodesCreated == 0 && snapshot.NodesUpdated == 0 {
		t.Fatalf("expected processor to record node mutations, got: %#v", snapshot)
	}
	if snapshot.RelationshipsCreated == 0 && snapshot.RelationshipsUpdated == 0 {
		t.Fatalf("expected processor to record relationship mutations, got: %#v", snapshot)
	}
}

func TestRepositoryApplyPlanCreatesRelationshipCrossProductForMultipleMatches(t *testing.T) {
	tc.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	neoContainer, neo4jURI := startNeo4jContainer(ctx, t)
	defer terminateContainer(context.Background(), t, neoContainer)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repository := waitForNeo4jRepository(ctx, t, config.EnvConfig{
		Neo4jURI:           neo4jURI,
		Neo4jDatabase:      "neo4j",
		Neo4jUsername:      "neo4j",
		Neo4jPassword:      neo4jPassword,
		Neo4jTimeout:       5 * time.Second,
		VerifyConnectivity: true,
	}, logger)
	defer func() {
		if err := repository.Close(context.Background()); err != nil {
			t.Fatalf("close repository: %v", err)
		}
	}()

	nodePlan := domain.MutationPlan{
		Nodes: []domain.GraphNode{
			{
				Types:          []string{"SourceDemo"},
				Name:           "source-a",
				TemplateHashes: []string{"source-v1"},
				UpdatePolicy:   domain.UpdatePolicyCreate,
				Properties: map[string]any{
					"name":            "source-a",
					"group":           "demo",
					"node_uid":        "source-a-uid",
					"template_hashes": []string{"source-v1"},
					"origin":          "auto",
				},
				UID: "source-a-uid",
			},
			{
				Types:          []string{"SourceDemo"},
				Name:           "source-b",
				TemplateHashes: []string{"source-v1"},
				UpdatePolicy:   domain.UpdatePolicyCreate,
				Properties: map[string]any{
					"name":            "source-b",
					"group":           "demo",
					"node_uid":        "source-b-uid",
					"template_hashes": []string{"source-v1"},
					"origin":          "auto",
				},
				UID: "source-b-uid",
			},
			{
				Types:          []string{"TargetDemo"},
				Name:           "target-1",
				TemplateHashes: []string{"target-v1"},
				UpdatePolicy:   domain.UpdatePolicyCreate,
				Properties: map[string]any{
					"name":            "target-1",
					"group":           "demo",
					"node_uid":        "target-1-uid",
					"template_hashes": []string{"target-v1"},
					"origin":          "auto",
				},
				UID: "target-1-uid",
			},
			{
				Types:          []string{"TargetDemo"},
				Name:           "target-2",
				TemplateHashes: []string{"target-v1"},
				UpdatePolicy:   domain.UpdatePolicyCreate,
				Properties: map[string]any{
					"name":            "target-2",
					"group":           "demo",
					"node_uid":        "target-2-uid",
					"template_hashes": []string{"target-v1"},
					"origin":          "auto",
				},
				UID: "target-2-uid",
			},
			{
				Types:          []string{"TargetDemo"},
				Name:           "target-3",
				TemplateHashes: []string{"target-v1"},
				UpdatePolicy:   domain.UpdatePolicyCreate,
				Properties: map[string]any{
					"name":            "target-3",
					"group":           "demo",
					"node_uid":        "target-3-uid",
					"template_hashes": []string{"target-v1"},
					"origin":          "auto",
				},
				UID: "target-3-uid",
			},
		},
	}

	if _, err := repository.ApplyPlan(ctx, nodePlan); err != nil {
		t.Fatalf("create seed nodes: %v", err)
	}

	relationshipPlan := domain.MutationPlan{
		Relationships: []domain.GraphRelationship{
			{
				Type:         "CONNECTS_TO",
				TemplateHash: "connects-to-v1",
				UpdatePolicy: domain.UpdatePolicyCreate,
				Source: domain.NodeSelector{
					Type: "SourceDemo",
					Attributes: map[string]any{
						"group": "demo",
					},
				},
				Target: domain.NodeSelector{
					Type: "TargetDemo",
					Attributes: map[string]any{
						"group": "demo",
					},
				},
				Properties: map[string]any{
					"template_hash": "connects-to-v1",
					"origin":        "auto",
				},
				UID: "connects-to-template-uid",
			},
		},
	}

	firstStats, err := repository.ApplyPlan(ctx, relationshipPlan)
	if err != nil {
		t.Fatalf("create fan-out relationships: %v", err)
	}
	if firstStats.RelationshipsCreated != 6 {
		t.Fatalf("expected 6 relationships to be created, got %#v", firstStats)
	}

	secondStats, err := repository.ApplyPlan(ctx, relationshipPlan)
	if err != nil {
		t.Fatalf("reapply fan-out relationships: %v", err)
	}
	if secondStats.RelationshipsSkipped != 6 {
		t.Fatalf("expected 6 relationships to be skipped on reapply, got %#v", secondStats)
	}

	neo4jDriver, err := driver.NewDriverWithContext(neo4jURI, driver.BasicAuth("neo4j", neo4jPassword, ""))
	if err != nil {
		t.Fatalf("create verification driver: %v", err)
	}
	defer func() {
		if err := neo4jDriver.Close(ctx); err != nil {
			t.Fatalf("close verification driver: %v", err)
		}
	}()

	session := neo4jDriver.NewSession(ctx, driver.SessionConfig{DatabaseName: "neo4j"})
	defer func() {
		if err := session.Close(ctx); err != nil {
			t.Fatalf("close verification session: %v", err)
		}
	}()

	relationshipCount := readCount(ctx, t, session, `
MATCH (:Entity:SourceDemo {group: "demo"})-[r:CONNECTS_TO {template_hash: $template_hash, origin: "auto"}]->(:Entity:TargetDemo {group: "demo"})
RETURN count(r) AS count
`, map[string]any{
		"template_hash": "connects-to-v1",
	})
	if relationshipCount != 6 {
		t.Fatalf("expected exactly 6 fan-out relationships, got %d", relationshipCount)
	}

	distinctUIDCount := readCount(ctx, t, session, `
MATCH (:Entity:SourceDemo {group: "demo"})-[r:CONNECTS_TO {template_hash: $template_hash, origin: "auto"}]->(:Entity:TargetDemo {group: "demo"})
RETURN count(DISTINCT r.rel_uid) AS count
`, map[string]any{
		"template_hash": "connects-to-v1",
	})
	if distinctUIDCount != 6 {
		t.Fatalf("expected 6 distinct relationship rel_uid values, got %d", distinctUIDCount)
	}
}

func TestRepositoryApplyPlanDoesNotDuplicateNodeUnderConcurrentWrites(t *testing.T) {
	tc.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	neoContainer, neo4jURI := startNeo4jContainer(ctx, t)
	defer terminateContainer(context.Background(), t, neoContainer)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repository := waitForNeo4jRepository(ctx, t, config.EnvConfig{
		Neo4jURI:           neo4jURI,
		Neo4jDatabase:      "neo4j",
		Neo4jUsername:      "neo4j",
		Neo4jPassword:      neo4jPassword,
		Neo4jTimeout:       5 * time.Second,
		VerifyConnectivity: true,
	}, logger)
	defer func() {
		if err := repository.Close(context.Background()); err != nil {
			t.Fatalf("close repository: %v", err)
		}
	}()

	plan := domain.MutationPlan{
		Nodes: []domain.GraphNode{
			{
				Types:          []string{"ConcurrentHost"},
				Name:           "cadecrk01cl01vm03",
				TemplateHashes: []string{"host-v1"},
				UpdatePolicy:   domain.UpdatePolicyMergeAtChange,
				Properties: map[string]any{
					"name":            "cadecrk01cl01vm03",
					"node_uid":        "27933f3e-2bc1-5383-b673-31e5a8d87433",
					"template_hashes": []string{"host-v1"},
					"origin":          "auto",
				},
				UID: "27933f3e-2bc1-5383-b673-31e5a8d87433",
			},
		},
	}

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for index := 0; index < 8; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repository.ApplyPlan(ctx, plan)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent apply failed: %v", err)
		}
	}

	neo4jDriver, err := driver.NewDriverWithContext(neo4jURI, driver.BasicAuth("neo4j", neo4jPassword, ""))
	if err != nil {
		t.Fatalf("create verification driver: %v", err)
	}
	defer func() {
		if err := neo4jDriver.Close(ctx); err != nil {
			t.Fatalf("close verification driver: %v", err)
		}
	}()

	session := neo4jDriver.NewSession(ctx, driver.SessionConfig{DatabaseName: "neo4j"})
	defer func() {
		if err := session.Close(ctx); err != nil {
			t.Fatalf("close verification session: %v", err)
		}
	}()

	count := readCount(ctx, t, session, `
MATCH (n:Entity:ConcurrentHost {name: $name, node_uid: $node_uid})
RETURN count(n) AS count
`, map[string]any{
		"name":     "cadecrk01cl01vm03",
		"node_uid": "27933f3e-2bc1-5383-b673-31e5a8d87433",
	})
	if count != 1 {
		t.Fatalf("expected exactly one concurrent host node, got %d", count)
	}
}

func startPrometheusContainer(ctx context.Context, t *testing.T) (tc.Container, string) {
	t.Helper()

	req := tc.ContainerRequest{
		Image:        getenvOrDefault("INTEGRATION_PROMETHEUS_IMAGE", defaultPrometheusImage),
		ExposedPorts: []string{"9090/tcp"},
		WaitingFor: wait.ForHTTP("/-/ready").
			WithPort("9090/tcp").
			WithStartupTimeout(90 * time.Second),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start prometheus container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("resolve prometheus host: %v", err)
	}

	port, err := container.MappedPort(ctx, "9090/tcp")
	if err != nil {
		t.Fatalf("resolve prometheus port: %v", err)
	}

	return container, fmt.Sprintf("http://%s:%s", host, port.Port())
}

func startNeo4jContainer(ctx context.Context, t *testing.T) (tc.Container, string) {
	t.Helper()

	req := tc.ContainerRequest{
		Image:        getenvOrDefault("INTEGRATION_NEO4J_IMAGE", defaultNeo4jImage),
		ExposedPorts: []string{"7687/tcp"},
		Env: map[string]string{
			"NEO4J_AUTH": "neo4j/" + neo4jPassword,
		},
		WaitingFor: wait.ForListeningPort("7687/tcp").
			WithStartupTimeout(2 * time.Minute),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start neo4j container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("resolve neo4j host: %v", err)
	}

	port, err := container.MappedPort(ctx, "7687/tcp")
	if err != nil {
		t.Fatalf("resolve neo4j port: %v", err)
	}

	return container, fmt.Sprintf("bolt://%s:%s", host, port.Port())
}

func waitForPrometheusDatapoints(ctx context.Context, t *testing.T, client *prometheus.Client, query string) []domain.Datapoint {
	t.Helper()

	deadline := time.Now().Add(90 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		datapoints, err := client.Query(ctx, query)
		if err == nil && len(datapoints) > 0 {
			return datapoints
		}
		lastErr = err
		time.Sleep(1 * time.Second)
	}

	t.Fatalf("prometheus query %q did not return datapoints in time, last error: %v", query, lastErr)
	return nil
}

func waitForNeo4jRepository(ctx context.Context, t *testing.T, env config.EnvConfig, logger *slog.Logger) *neo4jrepo.Repository {
	t.Helper()

	deadline := time.Now().Add(90 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		repository, err := neo4jrepo.NewRepository(ctx, env, logger)
		if err == nil {
			return repository
		}

		lastErr = err
		time.Sleep(1 * time.Second)
	}

	t.Fatalf("neo4j repository was not ready in time: %v", lastErr)
	return nil
}

func verifyGraphState(ctx context.Context, t *testing.T, neo4jURI string) {
	t.Helper()

	neo4jDriver, err := driver.NewDriverWithContext(neo4jURI, driver.BasicAuth("neo4j", neo4jPassword, ""))
	if err != nil {
		t.Fatalf("create verification driver: %v", err)
	}
	defer func() {
		if err := neo4jDriver.Close(ctx); err != nil {
			t.Fatalf("close verification driver: %v", err)
		}
	}()

	session := neo4jDriver.NewSession(ctx, driver.SessionConfig{DatabaseName: "neo4j"})
	defer func() {
		if err := session.Close(ctx); err != nil {
			t.Fatalf("close verification session: %v", err)
		}
	}()

	prometheusCount := readCount(ctx, t, session, `
MATCH (n:Entity:Prometheus {name: $name, origin: "auto"})
RETURN count(n) AS count
`, map[string]any{"name": "main_prometheus"})
	if prometheusCount != 1 {
		t.Fatalf("expected exactly one Prometheus node, got %d", prometheusCount)
	}

	buildCount := readCount(ctx, t, session, `
MATCH (n:Entity:BuildVersion {origin: "auto"})
RETURN count(n) AS count
`, nil)
	if buildCount != 1 {
		t.Fatalf("expected exactly one BuildVersion node, got %d", buildCount)
	}

	relationshipCount := readCount(ctx, t, session, `
MATCH (:Entity:Prometheus {name: $name})-[r:EXPOSES_BUILD {template_hash: $template_hash, origin: "auto"}]->(:Entity:BuildVersion)
RETURN count(r) AS count
`, map[string]any{
		"name":          "main_prometheus",
		"template_hash": "prometheus-exposes-build-v1",
	})
	if relationshipCount != 1 {
		t.Fatalf("expected exactly one EXPOSES_BUILD relationship, got %d", relationshipCount)
	}

	if emptyFieldCount := readCount(ctx, t, session, `
MATCH (n:Entity:Prometheus {name: $name, origin: "auto"})
WHERE n.node_uid IS NULL OR n.updated_at IS NULL OR n.created_at IS NULL
RETURN count(n) AS count
`, map[string]any{"name": "main_prometheus"}); emptyFieldCount != 0 {
		t.Fatalf("expected automatic node fields to be populated, got %d invalid nodes", emptyFieldCount)
	}

	if emptyFieldCount := readCount(ctx, t, session, `
MATCH (:Entity:Prometheus {name: $name})-[r:EXPOSES_BUILD {template_hash: $template_hash, origin: "auto"}]->(:Entity:BuildVersion)
WHERE r.rel_uid IS NULL OR r.updated_at IS NULL OR r.created_at IS NULL
RETURN count(r) AS count
`, map[string]any{
		"name":          "main_prometheus",
		"template_hash": "prometheus-exposes-build-v1",
	}); emptyFieldCount != 0 {
		t.Fatalf("expected automatic relationship fields to be populated, got %d invalid relationships", emptyFieldCount)
	}
}

func readCount(ctx context.Context, t *testing.T, session driver.SessionWithContext, query string, params map[string]any) int64 {
	t.Helper()

	value, err := session.ExecuteRead(ctx, func(tx driver.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		if !result.Next(ctx) {
			if err := result.Err(); err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("query returned no rows")
		}

		countValue, ok := result.Record().Get("count")
		if !ok {
			return nil, fmt.Errorf("query did not return count")
		}

		return countValue, result.Err()
	})
	if err != nil {
		t.Fatalf("read count: %v", err)
	}

	count, ok := value.(int64)
	if !ok {
		t.Fatalf("expected count to be int64, got %T", value)
	}

	return count
}

func terminateContainer(ctx context.Context, t *testing.T, container tc.Container) {
	t.Helper()
	if container == nil {
		return
	}
	if err := container.Terminate(ctx); err != nil {
		t.Fatalf("terminate container: %v", err)
	}
}

func getenvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
