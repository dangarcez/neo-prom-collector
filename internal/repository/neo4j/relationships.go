package neo4j

import (
	"context"
	"errors"
	"fmt"
	"time"

	driver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"neo_collector_go/internal/domain"
)

func (r *Repository) applyRelationship(ctx context.Context, tx driver.ManagedTransaction, relationship domain.GraphRelationship) (domain.PersistAction, error) {
	sourceIDs, err := r.findSelectorMatches(ctx, tx, relationship.Source)
	if err != nil {
		return "", err
	}
	if len(sourceIDs) == 0 {
		return domain.PersistActionSkipped, nil
	}
	if len(sourceIDs) > 1 {
		return "", fmt.Errorf("%w for relationship source %q", domain.ErrAmbiguousNodeMatch, relationship.Source.Type)
	}

	targetIDs, err := r.findSelectorMatches(ctx, tx, relationship.Target)
	if err != nil {
		return "", err
	}
	if len(targetIDs) == 0 {
		return domain.PersistActionSkipped, nil
	}
	if len(targetIDs) > 1 {
		return "", fmt.Errorf("%w for relationship target %q", domain.ErrAmbiguousNodeMatch, relationship.Target.Type)
	}

	existingCount, err := r.countRelationshipsByIdentity(ctx, tx, sourceIDs[0], targetIDs[0], relationship)
	if err != nil {
		return "", err
	}
	if existingCount > 1 {
		return "", errors.New("ambiguous equivalent relationship match")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	switch relationship.UpdatePolicy {
	case domain.UpdatePolicyCreate:
		if existingCount > 0 {
			return domain.PersistActionSkipped, nil
		}
		return r.createRelationship(ctx, tx, sourceIDs[0], targetIDs[0], relationship, now)
	case domain.UpdatePolicyMerge:
		if existingCount == 0 {
			return r.createRelationship(ctx, tx, sourceIDs[0], targetIDs[0], relationship, now)
		}
		return r.updateRelationship(ctx, tx, sourceIDs[0], targetIDs[0], relationship, now)
	default:
		return "", fmt.Errorf("unsupported relationship update policy %q", relationship.UpdatePolicy)
	}
}

func (r *Repository) findSelectorMatches(ctx context.Context, tx driver.ManagedTransaction, selector domain.NodeSelector) ([]string, error) {
	labels, err := labelsFragment([]string{selector.Type})
	if err != nil {
		return nil, err
	}

	whereClause, params, err := buildPropertyFilters("n", selector.Attributes, "selector")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("MATCH (n%s)", labels)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	query += " RETURN elementId(n) AS element_id LIMIT 2"

	result, err := tx.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("run selector query: %w", err)
	}

	ids := []string{}
	for result.Next(ctx) {
		value, ok := result.Record().Get("element_id")
		if !ok {
			return nil, fmt.Errorf("selector query did not return element_id")
		}

		id, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("selector element_id has unexpected type %T", value)
		}

		ids = append(ids, id)
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("consume selector query: %w", err)
	}

	return ids, nil
}

func (r *Repository) countRelationshipsByIdentity(
	ctx context.Context,
	tx driver.ManagedTransaction,
	sourceID string,
	targetID string,
	relationship domain.GraphRelationship,
) (int, error) {
	relationshipType, err := sanitizeIdentifier(relationship.Type)
	if err != nil {
		return 0, err
	}

	templateHashes := relationshipTemplateHashes(relationship)

	query := fmt.Sprintf(`
MATCH (source) WHERE elementId(source) = $source_id
MATCH (target) WHERE elementId(target) = $target_id
OPTIONAL MATCH (source)-[rel:%s {template_hashes: $template_hashes}]->(target)
RETURN count(rel) AS count
`, relationshipType)

	count, err := executeCountQuery(ctx, tx, query, map[string]any{
		"source_id":       sourceID,
		"target_id":       targetID,
		"template_hashes": templateHashes,
	}, "count")
	if err != nil {
		return 0, fmt.Errorf("count existing relationships: %w", err)
	}

	return count, nil
}

func (r *Repository) createRelationship(
	ctx context.Context,
	tx driver.ManagedTransaction,
	sourceID string,
	targetID string,
	relationship domain.GraphRelationship,
	now string,
) (domain.PersistAction, error) {
	relationshipType, err := sanitizeIdentifier(relationship.Type)
	if err != nil {
		return "", err
	}

	properties := cloneMap(relationship.Properties)
	properties["created_at"] = now
	properties["updated_at"] = now

	query := fmt.Sprintf(`
MATCH (source) WHERE elementId(source) = $source_id
MATCH (target) WHERE elementId(target) = $target_id
CREATE (source)-[rel:%s]->(target)
SET rel += $properties
RETURN 'created' AS action
`, relationshipType)

	return executeActionQuery(ctx, tx, query, map[string]any{
		"source_id":  sourceID,
		"target_id":  targetID,
		"properties": properties,
	})
}

func (r *Repository) updateRelationship(
	ctx context.Context,
	tx driver.ManagedTransaction,
	sourceID string,
	targetID string,
	relationship domain.GraphRelationship,
	now string,
) (domain.PersistAction, error) {
	relationshipType, err := sanitizeIdentifier(relationship.Type)
	if err != nil {
		return "", err
	}

	templateHashes := relationshipTemplateHashes(relationship)

	properties := cloneMap(relationship.Properties)
	properties["updated_at"] = now

	query := fmt.Sprintf(`
MATCH (source) WHERE elementId(source) = $source_id
MATCH (target) WHERE elementId(target) = $target_id
MATCH (source)-[rel:%s {template_hashes: $template_hashes}]->(target)
SET rel += $properties
RETURN 'updated' AS action
`, relationshipType)

	return executeActionQuery(ctx, tx, query, map[string]any{
		"source_id":       sourceID,
		"target_id":       targetID,
		"template_hashes": templateHashes,
		"properties":      properties,
	})
}

func relationshipTemplateHashes(relationship domain.GraphRelationship) []string {
	if value, ok := relationship.Properties["template_hashes"]; ok {
		switch typed := value.(type) {
		case []string:
			return append([]string(nil), typed...)
		case []any:
			hashes := make([]string, 0, len(typed))
			for _, item := range typed {
				hashes = append(hashes, fmt.Sprint(item))
			}
			return hashes
		}
	}
	if relationship.TemplateHash == "" {
		return nil
	}
	return []string{relationship.TemplateHash}
}
