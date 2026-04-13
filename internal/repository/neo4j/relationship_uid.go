package neo4j

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"neo_collector_go/internal/domain"
)

func relationshipForMatch(
	relationship domain.GraphRelationship,
	sourceMatch matchedNode,
	targetMatch matchedNode,
) domain.GraphRelationship {
	properties := cloneMap(relationship.Properties)
	properties["rel_uid"] = relationshipUIDForMatch(relationship, sourceMatch, targetMatch)
	relationship.Properties = properties
	return relationship
}

func relationshipUIDForMatch(
	relationship domain.GraphRelationship,
	sourceMatch matchedNode,
	targetMatch matchedNode,
) string {
	return stableRelationshipUUID(
		relationship.UID,
		matchedNodeIdentity(sourceMatch),
		matchedNodeIdentity(targetMatch),
	)
}

func matchedNodeIdentity(node matchedNode) string {
	if node.UID != "" {
		return "node_uid:" + node.UID
	}
	return "element_id:" + node.ElementID
}

func stableRelationshipUUID(parts ...string) string {
	hash := sha1.Sum([]byte(joinWithUnitSeparator(parts...)))
	raw := hash[:16]

	raw[6] = (raw[6] & 0x0f) | 0x50
	raw[8] = (raw[8] & 0x3f) | 0x80

	encoded := hex.EncodeToString(raw)
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		encoded[0:8],
		encoded[8:12],
		encoded[12:16],
		encoded[16:20],
		encoded[20:32],
	)
}

func joinWithUnitSeparator(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	joined := parts[0]
	for index := 1; index < len(parts); index++ {
		joined += "\x1f" + parts[index]
	}

	return joined
}
