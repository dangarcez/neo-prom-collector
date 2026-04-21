package neo4j

import (
	"context"
	"errors"
	"fmt"
	"time"

	driver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"neo_collector_go/internal/domain"
)

type matchedNode struct {
	ElementID string
	UID       string
}

func (r *Repository) applyRelationship(ctx context.Context, tx driver.ManagedTransaction, relationship domain.GraphRelationship) (domain.ApplyStats, error) {
	sourceMatches, err := r.findSelectorMatches(ctx, tx, relationship.Source)
	if err != nil {
		return domain.ApplyStats{}, err
	}
	if len(sourceMatches) == 0 {
		stats := domain.ApplyStats{}
		stats.AddRelationship(domain.PersistActionSkipped)
		return stats, nil
	}

	targetMatches, err := r.findSelectorMatches(ctx, tx, relationship.Target)
	if err != nil {
		return domain.ApplyStats{}, err
	}
	if len(targetMatches) == 0 {
		stats := domain.ApplyStats{}
		stats.AddRelationship(domain.PersistActionSkipped)
		return stats, nil
	}

	stats := domain.ApplyStats{}
	for _, sourceMatch := range sourceMatches {
		for _, targetMatch := range targetMatches {
			action, err := r.applyRelationshipMatch(ctx, tx, relationship, sourceMatch, targetMatch)
			if err != nil {
				return domain.ApplyStats{}, err
			}
			stats.AddRelationship(action)
		}
	}

	return stats, nil
}

func (r *Repository) applyRelationshipMatch(
	ctx context.Context,
	tx driver.ManagedTransaction,
	relationship domain.GraphRelationship,
	sourceMatch matchedNode,
	targetMatch matchedNode,
) (domain.PersistAction, error) {
	relationship = relationshipForMatch(relationship, sourceMatch, targetMatch)

	existingCount, err := r.countRelationshipsByIdentity(ctx, tx, sourceMatch.ElementID, targetMatch.ElementID, relationship)
	if err != nil {
		return "", err
	}
	if existingCount > 1 {
		return "", errors.New("ambiguous equivalent relationship match")
	}

	now := time.Now().UTC()

	switch relationship.UpdatePolicy {
	case domain.UpdatePolicyCreate:
		if existingCount > 0 {
			return domain.PersistActionSkipped, nil
		}
		return r.createRelationship(ctx, tx, sourceMatch.ElementID, targetMatch.ElementID, relationship, now)
	case domain.UpdatePolicyMerge:
		if existingCount == 0 {
			return r.createRelationship(ctx, tx, sourceMatch.ElementID, targetMatch.ElementID, relationship, now)
		}
		return r.updateRelationship(ctx, tx, sourceMatch.ElementID, targetMatch.ElementID, relationship, now)
	case domain.UpdatePolicyMergeAtChange:
		if existingCount == 0 {
			return r.createRelationship(ctx, tx, sourceMatch.ElementID, targetMatch.ElementID, relationship, now)
		}

		existingProperties, err := r.loadRelationshipPropertiesByIdentity(ctx, tx, sourceMatch.ElementID, targetMatch.ElementID, relationship)
		if err != nil {
			return "", err
		}
		if !shouldUpdateProperties(existingProperties, managedRelationshipProperties(relationship)) {
			return domain.PersistActionSkipped, nil
		}

		return r.updateRelationship(ctx, tx, sourceMatch.ElementID, targetMatch.ElementID, relationship, now)
	default:
		return "", fmt.Errorf("unsupported relationship update policy %q", relationship.UpdatePolicy)
	}
}

func (r *Repository) findSelectorMatches(ctx context.Context, tx driver.ManagedTransaction, selector domain.NodeSelector) ([]matchedNode, error) {
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
	query += " RETURN elementId(n) AS element_id, n.node_uid AS node_uid ORDER BY coalesce(n.node_uid, elementId(n))"

	result, err := tx.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("run selector query: %w", err)
	}

	matches := []matchedNode{}
	for result.Next(ctx) {
		value, ok := result.Record().Get("element_id")
		if !ok {
			return nil, fmt.Errorf("selector query did not return element_id")
		}

		id, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("selector element_id has unexpected type %T", value)
		}

		nodeUID := ""
		if uidValue, ok := result.Record().Get("node_uid"); ok && uidValue != nil {
			typedUID, ok := uidValue.(string)
			if !ok {
				return nil, fmt.Errorf("selector node_uid has unexpected type %T", uidValue)
			}
			nodeUID = typedUID
		}

		matches = append(matches, matchedNode{
			ElementID: id,
			UID:       nodeUID,
		})
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("consume selector query: %w", err)
	}

	return matches, nil
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
	now time.Time,
) (domain.PersistAction, error) {
	relationshipType, err := sanitizeIdentifier(relationship.Type)
	if err != nil {
		return "", err
	}

	properties := cloneMap(relationship.Properties)
	nowText := now.Format(time.RFC3339)
	properties["created_at"] = nowText
	properties["updated_at"] = nowText
	applyExpiration(properties, relationship.UpdatePolicy, relationship.ExpirationTimeMin, now)

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

func (r *Repository) loadRelationshipPropertiesByIdentity(
	ctx context.Context,
	tx driver.ManagedTransaction,
	sourceID string,
	targetID string,
	relationship domain.GraphRelationship,
) (map[string]any, error) {
	relationshipType, err := sanitizeIdentifier(relationship.Type)
	if err != nil {
		return nil, err
	}

	templateHashes := relationshipTemplateHashes(relationship)

	query := fmt.Sprintf(`
MATCH (source) WHERE elementId(source) = $source_id
MATCH (target) WHERE elementId(target) = $target_id
MATCH (source)-[rel:%s {template_hashes: $template_hashes}]->(target)
RETURN properties(rel) AS properties
LIMIT 1
`, relationshipType)

	result, err := tx.Run(ctx, query, map[string]any{
		"source_id":       sourceID,
		"target_id":       targetID,
		"template_hashes": templateHashes,
	})
	if err != nil {
		return nil, fmt.Errorf("run relationship properties query: %w", err)
	}

	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("relationship properties query returned no rows")
	}

	value, ok := result.Record().Get("properties")
	if !ok {
		return nil, fmt.Errorf("relationship properties query did not return properties")
	}

	properties, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("relationship properties have unexpected type %T", value)
	}

	if err := result.Err(); err != nil {
		return nil, err
	}

	return properties, nil
}

func (r *Repository) updateRelationship(
	ctx context.Context,
	tx driver.ManagedTransaction,
	sourceID string,
	targetID string,
	relationship domain.GraphRelationship,
	now time.Time,
) (domain.PersistAction, error) {
	relationshipType, err := sanitizeIdentifier(relationship.Type)
	if err != nil {
		return "", err
	}

	templateHashes := relationshipTemplateHashes(relationship)

	properties := cloneMap(relationship.Properties)
	properties["updated_at"] = now.Format(time.RFC3339)
	applyExpiration(properties, relationship.UpdatePolicy, relationship.ExpirationTimeMin, now)

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
