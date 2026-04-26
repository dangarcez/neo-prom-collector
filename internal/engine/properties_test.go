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
		nil,
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
		nil,
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

func TestResolvePropertiesAppliesPropertyTransforms(t *testing.T) {
	datapoint := domain.Datapoint{
		Labels: map[string]string{
			"pod": "Api-0",
		},
		Value:     1,
		Timestamp: time.Date(2026, 4, 11, 15, 4, 5, 0, time.UTC),
	}

	properties, err := ResolveProperties(
		map[string]any{
			"kind":     "WORKLOAD",
			"pipeline": "Api-Prod",
		},
		map[string]string{
			"name": "pod",
		},
		nil,
		[]config.PropertyTransformConfig{
			{
				Property: "name",
				Process: []config.PropertyProcessorConfig{
					{Type: config.PropertyProcessorTypeToUpper},
				},
			},
			{
				Property: "kind",
				Process: []config.PropertyProcessorConfig{
					{Type: config.PropertyProcessorTypeToLower},
				},
			},
			{
				Property: "pipeline",
				Process: []config.PropertyProcessorConfig{
					{Type: config.PropertyProcessorTypeToUpper},
					{Type: config.PropertyProcessorTypeToLower},
				},
			},
		},
		datapoint,
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if properties["name"] != "API-0" {
		t.Fatalf("expected transformed name, got: %#v", properties["name"])
	}
	if properties["kind"] != "workload" {
		t.Fatalf("expected transformed kind, got: %#v", properties["kind"])
	}
	if properties["pipeline"] != "api-prod" {
		t.Fatalf("expected processors to apply in order, got: %#v", properties["pipeline"])
	}
}

func TestResolvePropertiesIgnoresMissingOrNonStringPropertyTransforms(t *testing.T) {
	datapoint := domain.Datapoint{
		Labels: map[string]string{
			"pod": "api-0",
		},
		Value:     1,
		Timestamp: time.Date(2026, 4, 11, 15, 4, 5, 0, time.UTC),
	}

	properties, err := ResolveProperties(
		map[string]any{
			"attempts": 3,
		},
		map[string]string{
			"name": "pod",
		},
		nil,
		[]config.PropertyTransformConfig{
			{
				Property: "missing",
				Process: []config.PropertyProcessorConfig{
					{Type: config.PropertyProcessorTypeToUpper},
				},
			},
			{
				Property: "attempts",
				Process: []config.PropertyProcessorConfig{
					{Type: config.PropertyProcessorTypeToUpper},
				},
			},
		},
		datapoint,
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if properties["attempts"] != 3 {
		t.Fatalf("expected non-string property to remain unchanged, got: %#v", properties["attempts"])
	}
	if _, exists := properties["missing"]; exists {
		t.Fatalf("expected missing property transform target to remain absent, got: %#v", properties["missing"])
	}
}

func TestResolvePropertiesAppliesRegexPropertyTransform(t *testing.T) {
	datapoint := domain.Datapoint{
		Labels: map[string]string{
			"metric": "cpu_vru",
		},
		Value:     1,
		Timestamp: time.Date(2026, 4, 11, 15, 4, 5, 0, time.UTC),
	}

	properties, err := ResolveProperties(
		nil,
		map[string]string{
			"name": "metric",
		},
		nil,
		[]config.PropertyTransformConfig{
			{
				Property: "name",
				Process: []config.PropertyProcessorConfig{
					{
						Type:    config.PropertyProcessorTypeRegex,
						Pattern: `/(\w+)_(\w+)/`,
						Output:  "$1_and_$2",
					},
				},
			},
		},
		datapoint,
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if properties["name"] != "cpu_and_vru" {
		t.Fatalf("expected regex-transformed name, got: %#v", properties["name"])
	}
}

func TestResolvePropertiesIgnoresRegexPropertyTransformWithoutMatch(t *testing.T) {
	datapoint := domain.Datapoint{
		Labels: map[string]string{
			"metric": "cpu",
		},
		Value:     1,
		Timestamp: time.Date(2026, 4, 11, 15, 4, 5, 0, time.UTC),
	}

	properties, err := ResolveProperties(
		nil,
		map[string]string{
			"name": "metric",
		},
		nil,
		[]config.PropertyTransformConfig{
			{
				Property: "name",
				Process: []config.PropertyProcessorConfig{
					{
						Type:    config.PropertyProcessorTypeRegex,
						Pattern: `/(\w+)_(\w+)/`,
						Output:  "$1_and_$2",
					},
				},
			},
		},
		datapoint,
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if properties["name"] != "cpu" {
		t.Fatalf("expected unmatched regex to preserve value, got: %#v", properties["name"])
	}
}
