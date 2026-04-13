package engine

import (
	"testing"
	"time"

	"neo_collector_go/internal/config"
	"neo_collector_go/internal/domain"
)

func TestResolveProperties(t *testing.T) {
	datapoint := domain.Datapoint{
		Labels: map[string]string{
			"pod":   "api-0",
			"stage": "prod",
		},
		Value:     150,
		Timestamp: time.Date(2026, 4, 11, 15, 4, 5, 0, time.UTC),
	}

	properties, err := ResolveProperties(
		map[string]any{
			"kind": "pod",
		},
		map[string]string{
			"name":      "pod",
			"scrape":    "__value__",
			"timestamp": "__timestamp__",
		},
		[]config.ConditionalPropertyConfig{
			{
				Type:  "static",
				Name:  "activity",
				Value: "high",
				Conditions: []config.ConditionConfig{
					{
						Type:        "value",
						GreaterThan: float64Pointer(100),
					},
				},
			},
			{
				Type:      "label",
				Name:      "stage_name",
				FromLabel: "stage",
				Conditions: []config.ConditionConfig{
					{
						Type:   "label",
						Label:  "stage",
						Equals: "prod",
					},
				},
			},
		},
		datapoint,
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if properties["kind"] != "pod" {
		t.Fatalf("expected static property to be preserved, got: %#v", properties["kind"])
	}
	if properties["name"] != "api-0" {
		t.Fatalf("expected label property to be resolved, got: %#v", properties["name"])
	}
	if properties["activity"] != "high" {
		t.Fatalf("expected conditional static property to be applied, got: %#v", properties["activity"])
	}
	if properties["stage_name"] != "prod" {
		t.Fatalf("expected conditional label property to be applied, got: %#v", properties["stage_name"])
	}
	if properties["scrape"] != float64(150) {
		t.Fatalf("expected __value__ to resolve, got: %#v", properties["scrape"])
	}
	if properties["timestamp"] != "2026-04-11T15:04:05Z" {
		t.Fatalf("expected __timestamp__ to resolve, got: %#v", properties["timestamp"])
	}
}

func TestResolvePropertiesSkipsMissingLabelProperties(t *testing.T) {
	datapoint := domain.Datapoint{
		Labels: map[string]string{
			"pod": "api-0",
		},
		Value:     1,
		Timestamp: time.Date(2026, 4, 11, 15, 4, 5, 0, time.UTC),
	}

	properties, err := ResolveProperties(
		map[string]any{
			"kind": "pod",
		},
		map[string]string{
			"name":      "pod",
			"namespace": "namespace",
		},
		nil,
		datapoint,
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if properties["name"] != "api-0" {
		t.Fatalf("expected name to be resolved, got: %#v", properties["name"])
	}
	if _, exists := properties["namespace"]; exists {
		t.Fatalf("expected missing label property to be omitted, got: %#v", properties["namespace"])
	}
}

func TestResolvePropertiesSkipsMissingConditionalLabelProperty(t *testing.T) {
	datapoint := domain.Datapoint{
		Labels: map[string]string{
			"pod":   "api-0",
			"stage": "prod",
		},
		Value:     1,
		Timestamp: time.Date(2026, 4, 11, 15, 4, 5, 0, time.UTC),
	}

	properties, err := ResolveProperties(
		nil,
		map[string]string{
			"name": "pod",
		},
		[]config.ConditionalPropertyConfig{
			{
				Type:      "label",
				Name:      "team",
				FromLabel: "owner_team",
				Conditions: []config.ConditionConfig{
					{
						Type:   "label",
						Label:  "stage",
						Equals: "prod",
					},
				},
			},
		},
		datapoint,
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if properties["name"] != "api-0" {
		t.Fatalf("expected name to be resolved, got: %#v", properties["name"])
	}
	if _, exists := properties["team"]; exists {
		t.Fatalf("expected missing conditional label property to be omitted, got: %#v", properties["team"])
	}
}
