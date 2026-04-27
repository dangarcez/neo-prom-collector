package neo4j

import (
	"fmt"
	"sort"
	"sync"

	"neo_collector_go/internal/domain"
)

func (r *Repository) lockPlan(plan domain.MutationPlan) func() {
	nodeKeys := uniqueSortedNodeLockKeys(plan.Nodes)
	relationshipKeys := uniqueSortedRelationshipLockKeys(plan.Relationships)

	unlocks := make([]func(), 0, len(nodeKeys)+len(relationshipKeys))
	for _, key := range nodeKeys {
		unlocks = append(unlocks, lockKey(&r.nodeLocks, key))
	}
	for _, key := range relationshipKeys {
		unlocks = append(unlocks, lockKey(&r.relationshipLocks, key))
	}

	return func() {
		for index := len(unlocks) - 1; index >= 0; index-- {
			unlocks[index]()
		}
	}
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
		return domain.FieldNodeUID + ":" + node.UID
	}

	if value, ok := node.Properties[domain.FieldNodeUID]; ok && value != nil {
		if typed, ok := value.(string); ok && typed != "" {
			return domain.FieldNodeUID + ":" + typed
		}
	}

	return "node_name:" + node.Name
}

func relationshipLockKey(relationship domain.GraphRelationship) string {
	if value, ok := relationship.Properties[domain.FieldRelUID]; ok && value != nil {
		if typed, ok := value.(string); ok && typed != "" {
			return domain.FieldRelUID + ":" + typed
		}
	}

	if relationship.UID != "" {
		return domain.FieldRelUID + ":" + relationship.UID
	}

	return "relationship_type:" + relationship.Type
}

func uniqueSortedNodeLockKeys(nodes []domain.GraphNode) []string {
	keys := map[string]struct{}{}
	for _, node := range nodes {
		key := nodeLockKey(node)
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
	}
	return sortedKeys(keys)
}

func uniqueSortedRelationshipLockKeys(relationships []domain.GraphRelationship) []string {
	keys := map[string]struct{}{}
	for _, relationship := range relationships {
		key := relationshipLockKey(relationship)
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
	}
	return sortedKeys(keys)
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
