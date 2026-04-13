package neo4j

import (
	"context"
	"fmt"
	"time"

	driver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"neo_collector_go/internal/domain"
)

func (r *Repository) applyNode(ctx context.Context, tx driver.ManagedTransaction, node domain.GraphNode) (domain.PersistAction, error) {
	existingCount, err := r.countNodesByIdentity(ctx, tx, node)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	switch node.UpdatePolicy {
	case domain.UpdatePolicyCreate:
		if existingCount > 1 {
			return "", fmt.Errorf("%w for node %q", domain.ErrAmbiguousNodeMatch, node.Name)
		}
		if existingCount > 0 {
			return domain.PersistActionSkipped, nil
		}
		return r.createNode(ctx, tx, node, now)
	case domain.UpdatePolicyMerge:
		if existingCount == 0 {
			return r.createNode(ctx, tx, node, now)
		}
		if existingCount > 1 {
			return "", fmt.Errorf("%w for node %q", domain.ErrAmbiguousNodeMatch, node.Name)
		}
		return r.updateNode(ctx, tx, node, now)
	case domain.UpdatePolicyMergeAtChange:
		if existingCount == 0 {
			return r.createNode(ctx, tx, node, now)
		}
		if existingCount > 1 {
			return "", fmt.Errorf("%w for node %q", domain.ErrAmbiguousNodeMatch, node.Name)
		}

		existingProperties, err := r.loadNodePropertiesByIdentity(ctx, tx, node)
		if err != nil {
			return "", err
		}
		if !shouldUpdateProperties(existingProperties, managedNodeProperties(node)) {
			return domain.PersistActionSkipped, nil
		}

		return r.updateNode(ctx, tx, node, now)
	default:
		return "", fmt.Errorf("unsupported node update policy %q", node.UpdatePolicy)
	}
}

func (r *Repository) countNodesByIdentity(ctx context.Context, tx driver.ManagedTransaction, node domain.GraphNode) (int, error) {
	labels, err := labelsFragment(node.Types)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(
		"MATCH (n%s {name: $name}) RETURN count(n) AS count",
		labels,
	)

	count, err := executeCountQuery(ctx, tx, query, map[string]any{
		"name": node.Name,
	}, "count")
	if err != nil {
		return 0, fmt.Errorf("count existing nodes: %w", err)
	}

	return count, nil
}

func (r *Repository) createNode(ctx context.Context, tx driver.ManagedTransaction, node domain.GraphNode, now string) (domain.PersistAction, error) {
	labels, err := labelsFragment(node.Types)
	if err != nil {
		return "", err
	}

	properties := cloneMap(node.Properties)
	properties["created_at"] = now
	properties["updated_at"] = now

	query := fmt.Sprintf(
		"CREATE (n%s {name: $name}) SET n += $properties RETURN 'created' AS action",
		labels,
	)

	return executeActionQuery(ctx, tx, query, map[string]any{
		"name":       node.Name,
		"properties": properties,
	})
}

func (r *Repository) loadNodePropertiesByIdentity(
	ctx context.Context,
	tx driver.ManagedTransaction,
	node domain.GraphNode,
) (map[string]any, error) {
	labels, err := labelsFragment(node.Types)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		"MATCH (n%s {name: $name}) RETURN properties(n) AS properties LIMIT 1",
		labels,
	)

	result, err := tx.Run(ctx, query, map[string]any{
		"name": node.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("run node properties query: %w", err)
	}

	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("node properties query returned no rows")
	}

	value, ok := result.Record().Get("properties")
	if !ok {
		return nil, fmt.Errorf("node properties query did not return properties")
	}

	properties, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("node properties have unexpected type %T", value)
	}

	if err := result.Err(); err != nil {
		return nil, err
	}

	return properties, nil
}

func (r *Repository) updateNode(ctx context.Context, tx driver.ManagedTransaction, node domain.GraphNode, now string) (domain.PersistAction, error) {
	labels, err := labelsFragment(node.Types)
	if err != nil {
		return "", err
	}

	properties := cloneMap(node.Properties)
	properties["updated_at"] = now

	query := fmt.Sprintf(
		"MATCH (n%s {name: $name}) SET n += $properties RETURN 'updated' AS action",
		labels,
	)

	return executeActionQuery(ctx, tx, query, map[string]any{
		"name":       node.Name,
		"properties": properties,
	})
}
