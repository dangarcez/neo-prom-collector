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
