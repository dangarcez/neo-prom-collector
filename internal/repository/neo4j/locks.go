package neo4j

import (
	"fmt"
	"sync"

	"neo_collector_go/internal/domain"
)

func (r *Repository) lockNode(node domain.GraphNode) func() {
	return lockKey(&r.nodeLocks, nodeLockKey(node))
}

func (r *Repository) lockRelationship(relationship domain.GraphRelationship) func() {
	return lockKey(&r.relationshipLocks, relationshipLockKey(relationship))
}

func lockKey(store *sync.Map, key string) func() {
	if key == "" {
		return func() {}
	}

	value, _ := store.LoadOrStore(key, &sync.Mutex{})
	mutex, ok := value.(*sync.Mutex)
	if !ok {
		panic(fmt.Sprintf("unexpected lock type %T", value))
	}

	mutex.Lock()
	return mutex.Unlock
}

func nodeLockKey(node domain.GraphNode) string {
	if node.UID != "" {
		return "node_uid:" + node.UID
	}

	if value, ok := node.Properties["node_uid"]; ok && value != nil {
		if typed, ok := value.(string); ok && typed != "" {
			return "node_uid:" + typed
		}
	}

	return "node_name:" + node.Name
}

func relationshipLockKey(relationship domain.GraphRelationship) string {
	if value, ok := relationship.Properties["rel_uid"]; ok && value != nil {
		if typed, ok := value.(string); ok && typed != "" {
			return "rel_uid:" + typed
		}
	}

	if relationship.UID != "" {
		return "rel_uid:" + relationship.UID
	}

	return "relationship_type:" + relationship.Type
}
