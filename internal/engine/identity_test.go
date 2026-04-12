package engine

import (
	"testing"

	"neo_collector_go/internal/domain"
)

func TestNodeUIDIsStableAcrossInputOrder(t *testing.T) {
	first := NodeUID([]string{"Pod", "EntityPod"}, "api-0", []string{"b", "a"})
	second := NodeUID([]string{"EntityPod", "Pod"}, "api-0", []string{"a", "b"})

	if first != second {
		t.Fatalf("expected node uid to be stable, got %s and %s", first, second)
	}
}

func TestRelationshipUIDIsStableAcrossSelectorMapOrder(t *testing.T) {
	sourceA := domain.NodeSelector{
		Type: "Namespace",
		Attributes: map[string]any{
			"name": "prod",
			"kind": "namespace",
		},
	}
	sourceB := domain.NodeSelector{
		Type: "Namespace",
		Attributes: map[string]any{
			"kind": "namespace",
			"name": "prod",
		},
	}

	target := domain.NodeSelector{
		Type: "Pod",
		Attributes: map[string]any{
			"name": "api-0",
		},
	}

	first := RelationshipUID("OWNS", "owns-v1", sourceA, target)
	second := RelationshipUID("OWNS", "owns-v1", sourceB, target)

	if first != second {
		t.Fatalf("expected relationship uid to be stable, got %s and %s", first, second)
	}
}
