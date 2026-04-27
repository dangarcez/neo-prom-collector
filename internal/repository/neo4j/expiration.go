package neo4j

import (
	"time"

	"neo_collector_go/internal/domain"
)

func applyExpiration(properties map[string]any, updatePolicy domain.UpdatePolicy, expirationTimeMin *int, now time.Time) {
	if !shouldSetExpiration(updatePolicy, expirationTimeMin) {
		return
	}

	expiresAt := now.UTC().Add(time.Duration(*expirationTimeMin) * time.Minute).Format(time.RFC3339)
	properties[domain.FieldExpiresAt] = expiresAt
}

func shouldSetExpiration(updatePolicy domain.UpdatePolicy, expirationTimeMin *int) bool {
	if expirationTimeMin == nil {
		return false
	}

	switch updatePolicy {
	case domain.UpdatePolicyCreate, domain.UpdatePolicyMerge:
		return true
	default:
		return false
	}
}
