package neo4j

import (
	"testing"
	"time"

	"neo_collector_go/internal/domain"
)

func TestApplyExpirationSetsExpiresAtForCreateAndMerge(t *testing.T) {
	now := time.Date(2026, 4, 21, 12, 30, 0, 0, time.UTC)
	minutes := 45

	createProperties := map[string]any{}
	applyExpiration(createProperties, domain.UpdatePolicyCreate, &minutes, now)
	if createProperties[domain.FieldExpiresAt] != "2026-04-21T13:15:00Z" {
		t.Fatalf("expected create z4j_expires_at to be set, got %#v", createProperties[domain.FieldExpiresAt])
	}

	mergeProperties := map[string]any{}
	applyExpiration(mergeProperties, domain.UpdatePolicyMerge, &minutes, now)
	if mergeProperties[domain.FieldExpiresAt] != "2026-04-21T13:15:00Z" {
		t.Fatalf("expected merge z4j_expires_at to be set, got %#v", mergeProperties[domain.FieldExpiresAt])
	}
}

func TestApplyExpirationSkipsMergeAtChangeAndMissingConfiguration(t *testing.T) {
	now := time.Date(2026, 4, 21, 12, 30, 0, 0, time.UTC)
	minutes := 45

	mergeAtChangeProperties := map[string]any{}
	applyExpiration(mergeAtChangeProperties, domain.UpdatePolicyMergeAtChange, &minutes, now)
	if _, exists := mergeAtChangeProperties[domain.FieldExpiresAt]; exists {
		t.Fatalf("expected merge_at_change to skip z4j_expires_at, got %#v", mergeAtChangeProperties[domain.FieldExpiresAt])
	}

	missingConfigProperties := map[string]any{}
	applyExpiration(missingConfigProperties, domain.UpdatePolicyCreate, nil, now)
	if _, exists := missingConfigProperties[domain.FieldExpiresAt]; exists {
		t.Fatalf("expected missing expiration configuration to skip z4j_expires_at, got %#v", missingConfigProperties[domain.FieldExpiresAt])
	}
}
