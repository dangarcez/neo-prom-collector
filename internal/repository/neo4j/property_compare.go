package neo4j

import (
	"reflect"

	"neo_collector_go/internal/domain"
)

var automaticNodeProperties = map[string]struct{}{
	"node_uid":        {},
	"template_hashes": {},
	"origin":          {},
	"created_at":      {},
	"updated_at":      {},
	"expires_at":      {},
}

var automaticRelationshipProperties = map[string]struct{}{
	"rel_uid":         {},
	"template_hash":   {},
	"template_hashes": {},
	"origin":          {},
	"created_at":      {},
	"updated_at":      {},
	"expires_at":      {},
}

func managedNodeProperties(node domain.GraphNode) map[string]any {
	return filterManagedProperties(node.Properties, automaticNodeProperties)
}

func managedRelationshipProperties(relationship domain.GraphRelationship) map[string]any {
	return filterManagedProperties(relationship.Properties, automaticRelationshipProperties)
}

func filterManagedProperties(properties map[string]any, excludedKeys map[string]struct{}) map[string]any {
	filtered := make(map[string]any, len(properties))
	for key, value := range properties {
		if _, excluded := excludedKeys[key]; excluded {
			continue
		}
		filtered[key] = value
	}
	return filtered
}

func shouldUpdateProperties(current map[string]any, desired map[string]any) bool {
	for key, desiredValue := range desired {
		currentValue, ok := current[key]
		if !ok {
			return true
		}

		if !propertyValuesEqual(currentValue, desiredValue) {
			return true
		}
	}

	return false
}

func propertyValuesEqual(left any, right any) bool {
	return reflect.DeepEqual(normalizeComparableValue(left), normalizeComparableValue(right))
}

func normalizeComparableValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case string, bool:
		return typed
	case int:
		return float64(typed)
	case int8:
		return float64(typed)
	case int16:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case uint:
		return float64(typed)
	case uint8:
		return float64(typed)
	case uint16:
		return float64(typed)
	case uint32:
		return float64(typed)
	case uint64:
		return float64(typed)
	case float32:
		return float64(typed)
	case float64:
		return typed
	case []any:
		normalized := make([]any, len(typed))
		for index, item := range typed {
			normalized[index] = normalizeComparableValue(item)
		}
		return normalized
	case []string:
		normalized := make([]any, len(typed))
		for index, item := range typed {
			normalized[index] = normalizeComparableValue(item)
		}
		return normalized
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, item := range typed {
			normalized[key] = normalizeComparableValue(item)
		}
		return normalized
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return nil
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		normalized := make([]any, rv.Len())
		for index := 0; index < rv.Len(); index++ {
			normalized[index] = normalizeComparableValue(rv.Index(index).Interface())
		}
		return normalized
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return value
		}

		normalized := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			normalized[iter.Key().String()] = normalizeComparableValue(iter.Value().Interface())
		}
		return normalized
	default:
		return value
	}
}
