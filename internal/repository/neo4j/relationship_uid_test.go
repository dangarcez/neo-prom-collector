package neo4j

import (
	"testing"

	"neo_collector_go/internal/domain"
)

func TestRelationshipUIDForMatchIsStable(t *testing.T) {
	relationship := domain.GraphRelationship{
		UID: "relationship-template-uid",
	}
	sourceMatch := matchedNode{
		ElementID: "source-element-1",
		UID:       "source-node-uid",
	}
	targetMatch := matchedNode{
		ElementID: "target-element-1",
		UID:       "target-node-uid",
	}

	first := relationshipUIDForMatch(relationship, sourceMatch, targetMatch)
	second := relationshipUIDForMatch(relationship, sourceMatch, targetMatch)

	if first != second {
		t.Fatalf("expected relationship uid to be stable, got %q and %q", first, second)
	}
}

func TestRelationshipUIDForMatchChangesAcrossPairs(t *testing.T) {
	relationship := domain.GraphRelationship{
		UID: "relationship-template-uid",
	}
	sourceA := matchedNode{
		ElementID: "source-element-1",
		UID:       "source-node-uid-1",
	}
	sourceB := matchedNode{
		ElementID: "source-element-2",
		UID:       "source-node-uid-2",
	}
	target := matchedNode{
		ElementID: "target-element-1",
		UID:       "target-node-uid",
	}

	first := relationshipUIDForMatch(relationship, sourceA, target)
	second := relationshipUIDForMatch(relationship, sourceB, target)

	if first == second {
		t.Fatalf("expected relationship uid to differ across matched pairs")
	}
}

func TestRelationshipUIDForMatchFallsBackToElementID(t *testing.T) {
	relationship := domain.GraphRelationship{
		UID: "relationship-template-uid",
	}
	sourceMatch := matchedNode{
		ElementID: "source-element-1",
	}
	targetMatch := matchedNode{
		ElementID: "target-element-1",
	}

	uid := relationshipUIDForMatch(relationship, sourceMatch, targetMatch)
	if uid == "" {
		t.Fatal("expected relationship uid to be generated even without z4j_node_uid")
	}
}
