package neo4j

import (
	"testing"

	"neo_collector_go/internal/domain"
)

func TestManagedNodePropertiesExcludeAutomaticFields(t *testing.T) {
	node := domain.GraphNode{
		Properties: map[string]any{
			"name":            "api-0",
			"kind":            "workload",
			"node_uid":        "node-1",
			"template_hashes": []string{"pod-v1"},
			"origin":          "auto",
			"created_at":      "2026-04-12T10:00:00Z",
			"updated_at":      "2026-04-12T10:00:00Z",
			"expires_at":      "2026-04-12T11:00:00Z",
		},
	}

	properties := managedNodeProperties(node)

	if len(properties) != 2 {
		t.Fatalf("expected only business properties to remain, got %#v", properties)
	}
	if properties["name"] != "api-0" {
		t.Fatalf("expected name to remain, got %#v", properties["name"])
	}
	if properties["kind"] != "workload" {
		t.Fatalf("expected kind to remain, got %#v", properties["kind"])
	}
}

func TestManagedRelationshipPropertiesExcludeAutomaticFields(t *testing.T) {
	relationship := domain.GraphRelationship{
		Properties: map[string]any{
			"source_system":   "prometheus",
			"status":          "up",
			"rel_uid":         "rel-1",
			"template_hashes": []string{"scrapes-v1"},
			"origin":          "auto",
			"created_at":      "2026-04-12T10:00:00Z",
			"updated_at":      "2026-04-12T10:00:00Z",
			"expires_at":      "2026-04-12T11:00:00Z",
		},
	}

	properties := managedRelationshipProperties(relationship)

	if len(properties) != 2 {
		t.Fatalf("expected only business properties to remain, got %#v", properties)
	}
	if properties["source_system"] != "prometheus" {
		t.Fatalf("expected source_system to remain, got %#v", properties["source_system"])
	}
	if properties["status"] != "up" {
		t.Fatalf("expected status to remain, got %#v", properties["status"])
	}
}

func TestShouldUpdatePropertiesReturnsFalseWhenDesiredBusinessFieldsAreEqual(t *testing.T) {
	current := map[string]any{
		"name":       "api-0",
		"kind":       "workload",
		"replicas":   int64(3),
		"labels":     []any{"blue", "stable"},
		"updated_at": "2026-04-12T10:00:00Z",
		"extra":      "kept",
	}
	desired := map[string]any{
		"name":     "api-0",
		"kind":     "workload",
		"replicas": 3,
		"labels":   []string{"blue", "stable"},
	}

	if shouldUpdateProperties(current, desired) {
		t.Fatalf("expected comparison to skip update when desired properties are unchanged")
	}
}

func TestShouldUpdatePropertiesReturnsTrueWhenDesiredBusinessFieldsChange(t *testing.T) {
	current := map[string]any{
		"name":   "api-0",
		"status": "up",
	}
	desired := map[string]any{
		"name":   "api-0",
		"status": "down",
	}

	if !shouldUpdateProperties(current, desired) {
		t.Fatalf("expected comparison to request update when a desired property changes")
	}
}
