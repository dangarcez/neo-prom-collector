package engine

import (
	"testing"
	"time"

	"neo_collector_go/internal/config"
	"neo_collector_go/internal/domain"
)

func TestMatchConditions(t *testing.T) {
	datapoint := domain.Datapoint{
		Labels: map[string]string{
			"namespace": "production",
		},
		Value:     150,
		Timestamp: time.Unix(1700000000, 0).UTC(),
	}

	match, err := MatchConditions([]config.ConditionConfig{
		{
			Type:   "label",
			Label:  "namespace",
			Equals: "production",
		},
		{
			Type:        "value",
			GreaterThan: float64Pointer(100),
		},
	}, datapoint)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !match {
		t.Fatal("expected conditions to match")
	}
}

func TestMatchConditionsReturnsFalseWhenConditionDoesNotMatch(t *testing.T) {
	datapoint := domain.Datapoint{
		Labels: map[string]string{
			"namespace": "production",
		},
		Value: 10,
	}

	match, err := MatchConditions([]config.ConditionConfig{
		{
			Type:   "label",
			Label:  "namespace",
			Equals: "staging",
		},
	}, datapoint)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if match {
		t.Fatal("expected conditions not to match")
	}
}
