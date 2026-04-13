package neo4j

import (
	"testing"

	"neo_collector_go/internal/domain"
)

func TestNodeLockKeyPrefersNodeUID(t *testing.T) {
	node := domain.GraphNode{
		Name: "db01",
		UID:  "node-uid-1",
		Properties: map[string]any{
			"node_uid": "node-uid-2",
		},
	}

	key := nodeLockKey(node)
	if key != "node_uid:node-uid-1" {
		t.Fatalf("expected lock key to use node UID, got %q", key)
	}
}

func TestRelationshipLockKeyUsesRelationshipUID(t *testing.T) {
	relationship := domain.GraphRelationship{
		Type: "HOSTED_AT",
		Properties: map[string]any{
			"rel_uid": "rel-uid-1",
		},
	}

	key := relationshipLockKey(relationship)
	if key != "rel_uid:rel-uid-1" {
		t.Fatalf("expected lock key to use rel_uid, got %q", key)
	}
}

func TestUniqueSortedNodeLockKeysDeduplicate(t *testing.T) {
	nodes := []domain.GraphNode{
		{
			UID: "node-a",
		},
		{
			UID: "node-a",
		},
		{
			UID: "node-b",
		},
	}

	keys := uniqueSortedNodeLockKeys(nodes)
	if len(keys) != 2 {
		t.Fatalf("expected 2 unique node keys, got %#v", keys)
	}
	if keys[0] != "node_uid:node-a" || keys[1] != "node_uid:node-b" {
		t.Fatalf("unexpected node keys order: %#v", keys)
	}
}

func TestUniqueSortedRelationshipLockKeysDeduplicate(t *testing.T) {
	relationships := []domain.GraphRelationship{
		{
			UID: "rel-b",
		},
		{
			UID: "rel-a",
		},
		{
			UID: "rel-a",
		},
	}

	keys := uniqueSortedRelationshipLockKeys(relationships)
	if len(keys) != 2 {
		t.Fatalf("expected 2 unique relationship keys, got %#v", keys)
	}
	if keys[0] != "rel_uid:rel-a" || keys[1] != "rel_uid:rel-b" {
		t.Fatalf("unexpected relationship keys order: %#v", keys)
	}
}
