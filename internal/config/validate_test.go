package config

import (
	"strings"
	"testing"

	"neo_collector_go/internal/domain"
)

func TestValidateFileConfigAcceptsMinimalConfig(t *testing.T) {
	cfg := validFileConfig()
	cfg.Normalize()

	if err := ValidateFileConfig(cfg); err != nil {
		t.Fatalf("expected config to be valid, got error: %v", err)
	}
}

func TestValidateFileConfigRejectsNodeWithoutName(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].LabelProperties = map[string]string{}
	cfg.Normalize()

	err := ValidateFileConfig(cfg)
	if err == nil {
		t.Fatal("expected validation to fail when node has no name property")
	}

	if !strings.Contains(err.Error(), "name property") {
		t.Fatalf("expected name property error, got: %v", err)
	}
}

func TestValidateFileConfigAcceptsMergeAtChangePolicy(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].UpdatePolicy = "merge_at_change"
	cfg.PromTargets[0].Jobs[0].Relationships[0].UpdatePolicy = "mergeAtChange"
	cfg.Normalize()

	if err := ValidateFileConfig(cfg); err != nil {
		t.Fatalf("expected config to accept merge_at_change, got error: %v", err)
	}
}

func TestValidateFileConfigAcceptsLabelExistsCondition(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].Conditions = []ConditionConfig{
		{
			Type:  "label_exists",
			Label: "namespace",
		},
	}
	cfg.Normalize()

	if err := ValidateFileConfig(cfg); err != nil {
		t.Fatalf("expected config to accept label_exists, got error: %v", err)
	}
}

func TestValidateFileConfigAcceptsPropertyTransforms(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms = []PropertyTransformConfig{
		{
			Property: "name",
			Process: []PropertyProcessorConfig{
				{Type: "to_upper"},
				{
					Type:    "regex",
					Pattern: `/(\w+)_(\w+)/`,
					Output:  "$1_and_$2",
				},
			},
		},
	}
	cfg.PromTargets[0].Jobs[0].Relationships[0].PropertyTransforms = []PropertyTransformConfig{
		{
			Property: "environment",
			Process: []PropertyProcessorConfig{
				{Type: "To_Lower"},
			},
		},
	}
	cfg.Normalize()

	if got := cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms[0].Process[0].Type; got != PropertyProcessorTypeToUpper {
		t.Fatalf("expected node processor type to normalize to %q, got %q", PropertyProcessorTypeToUpper, got)
	}
	if got := cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms[0].Process[1].Type; got != PropertyProcessorTypeRegex {
		t.Fatalf("expected node regex processor type to normalize to %q, got %q", PropertyProcessorTypeRegex, got)
	}
	if got := cfg.PromTargets[0].Jobs[0].Relationships[0].PropertyTransforms[0].Process[0].Type; got != PropertyProcessorTypeToLower {
		t.Fatalf("expected relationship processor type to normalize to %q, got %q", PropertyProcessorTypeToLower, got)
	}

	if err := ValidateFileConfig(cfg); err != nil {
		t.Fatalf("expected config to accept property_transforms, got error: %v", err)
	}
}

func TestValidateFileConfigRejectsRegexPropertyTransformWithoutPattern(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms = []PropertyTransformConfig{
		{
			Property: "name",
			Process: []PropertyProcessorConfig{
				{
					Type:   PropertyProcessorTypeRegex,
					Output: "$1",
				},
			},
		},
	}
	cfg.Normalize()

	err := ValidateFileConfig(cfg)
	if err == nil {
		t.Fatal("expected validation to fail when regex property transform has no pattern")
	}

	if !strings.Contains(err.Error(), "pattern is required") {
		t.Fatalf("expected pattern required error, got: %v", err)
	}
}

func TestValidateFileConfigRejectsRegexPropertyTransformWithoutOutput(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms = []PropertyTransformConfig{
		{
			Property: "name",
			Process: []PropertyProcessorConfig{
				{
					Type:    PropertyProcessorTypeRegex,
					Pattern: `/(\w+)_(\w+)/`,
				},
			},
		},
	}
	cfg.Normalize()

	err := ValidateFileConfig(cfg)
	if err == nil {
		t.Fatal("expected validation to fail when regex property transform has no output")
	}

	if !strings.Contains(err.Error(), "output is required") {
		t.Fatalf("expected output required error, got: %v", err)
	}
}

func TestValidateFileConfigRejectsRegexPropertyTransformWithoutGroups(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms = []PropertyTransformConfig{
		{
			Property: "name",
			Process: []PropertyProcessorConfig{
				{
					Type:    PropertyProcessorTypeRegex,
					Pattern: `\w+_\w+`,
					Output:  "$1",
				},
			},
		},
	}
	cfg.Normalize()

	err := ValidateFileConfig(cfg)
	if err == nil {
		t.Fatal("expected validation to fail when regex property transform has no capture groups")
	}

	if !strings.Contains(err.Error(), "capture group") {
		t.Fatalf("expected capture group error, got: %v", err)
	}
}

func TestValidateFileConfigRejectsRegexPropertyTransformWithoutOutputGroupReference(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms = []PropertyTransformConfig{
		{
			Property: "name",
			Process: []PropertyProcessorConfig{
				{
					Type:    PropertyProcessorTypeRegex,
					Pattern: `/(\w+)_(\w+)/`,
					Output:  "constant",
				},
			},
		},
	}
	cfg.Normalize()

	err := ValidateFileConfig(cfg)
	if err == nil {
		t.Fatal("expected validation to fail when regex output does not reference a capture group")
	}

	if !strings.Contains(err.Error(), "at least one capture group") {
		t.Fatalf("expected output group reference error, got: %v", err)
	}
}

func TestValidateFileConfigRejectsRegexPropertyTransformWithUnknownGroupReference(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms = []PropertyTransformConfig{
		{
			Property: "name",
			Process: []PropertyProcessorConfig{
				{
					Type:    PropertyProcessorTypeRegex,
					Pattern: `/(\w+)_(\w+)/`,
					Output:  "$1_and_$3",
				},
			},
		},
	}
	cfg.Normalize()

	err := ValidateFileConfig(cfg)
	if err == nil {
		t.Fatal("expected validation to fail when regex output references an unknown group")
	}

	if !strings.Contains(err.Error(), "references $3") {
		t.Fatalf("expected group reference error, got: %v", err)
	}
}

func TestValidateFileConfigAcceptsExpirationTimeMin(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].ExpirationTimeMin = intPointer(30)
	cfg.PromTargets[0].Jobs[0].Relationships[0].ExpirationTimeMin = intPointer(15)
	cfg.Normalize()

	if err := ValidateFileConfig(cfg); err != nil {
		t.Fatalf("expected config to accept expiration_time_min, got error: %v", err)
	}
}

func TestValidateFileConfigRejectsNonPositiveExpirationTimeMin(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].ExpirationTimeMin = intPointer(0)
	cfg.Normalize()

	err := ValidateFileConfig(cfg)
	if err == nil {
		t.Fatal("expected validation to fail for non-positive expiration_time_min")
	}

	if !strings.Contains(err.Error(), "expiration_time_min") {
		t.Fatalf("expected expiration_time_min error, got: %v", err)
	}
}

func TestValidateFileConfigRejectsPropertyTransformWithoutProperty(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms = []PropertyTransformConfig{
		{
			Process: []PropertyProcessorConfig{
				{Type: PropertyProcessorTypeToUpper},
			},
		},
	}
	cfg.Normalize()

	err := ValidateFileConfig(cfg)
	if err == nil {
		t.Fatal("expected validation to fail when property_transforms item has no property")
	}

	if !strings.Contains(err.Error(), "property is required") {
		t.Fatalf("expected property required error, got: %v", err)
	}
}

func TestValidateFileConfigRejectsPropertyTransformWithoutProcess(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms = []PropertyTransformConfig{
		{
			Property: "name",
		},
	}
	cfg.Normalize()

	err := ValidateFileConfig(cfg)
	if err == nil {
		t.Fatal("expected validation to fail when property_transforms item has no processors")
	}

	if !strings.Contains(err.Error(), "must define at least one processor") {
		t.Fatalf("expected process required error, got: %v", err)
	}
}

func TestValidateFileConfigRejectsUnknownPropertyProcessorType(t *testing.T) {
	cfg := validFileConfig()
	cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms = []PropertyTransformConfig{
		{
			Property: "name",
			Process: []PropertyProcessorConfig{
				{Type: "trim"},
			},
		},
	}
	cfg.Normalize()

	err := ValidateFileConfig(cfg)
	if err == nil {
		t.Fatal("expected validation to fail for unknown property processor type")
	}

	if !strings.Contains(err.Error(), ".process[0].type") {
		t.Fatalf("expected processor type path in error, got: %v", err)
	}
}

func TestValidateFileConfigRejectsReservedPropertyPrefix(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*FileConfig)
	}{
		{
			name: "static property",
			mutate: func(cfg *FileConfig) {
				cfg.PromTargets[0].Jobs[0].Nodes[0].StaticProperties = map[string]any{
					domain.AppFieldPrefix + "custom": "value",
				}
			},
		},
		{
			name: "label property",
			mutate: func(cfg *FileConfig) {
				cfg.PromTargets[0].Jobs[0].Nodes[0].LabelProperties[domain.AppFieldPrefix+"custom"] = "pod"
			},
		},
		{
			name: "conditional property",
			mutate: func(cfg *FileConfig) {
				cfg.PromTargets[0].Jobs[0].Nodes[0].ConditionalProperties = []ConditionalPropertyConfig{
					{
						Name:  domain.AppFieldPrefix + "custom",
						Type:  "static",
						Value: "value",
						Conditions: []ConditionConfig{
							{
								Type:  "label_exists",
								Label: "pod",
							},
						},
					},
				}
			},
		},
		{
			name: "property transform",
			mutate: func(cfg *FileConfig) {
				cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms = []PropertyTransformConfig{
					{
						Property: domain.AppFieldPrefix + "custom",
						Process: []PropertyProcessorConfig{
							{Type: PropertyProcessorTypeToUpper},
						},
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validFileConfig()
			tt.mutate(&cfg)
			cfg.Normalize()

			err := ValidateFileConfig(cfg)
			if err == nil {
				t.Fatal("expected validation to fail for reserved property prefix")
			}
			if !strings.Contains(err.Error(), "reserved prefix") {
				t.Fatalf("expected reserved prefix error, got: %v", err)
			}
		})
	}
}

func validFileConfig() FileConfig {
	return FileConfig{
		PromTargets: []PromTargetConfig{
			{
				Name:           "prom",
				BaseURL:        "http://localhost:9090",
				TimeoutSeconds: 10,
				Runtime: TargetRuntimeConfig{
					DefaultIntervalSeconds: 30,
				},
				Jobs: []JobConfig{
					{
						Name:            "job",
						Query:           "up",
						IntervalSeconds: 30,
						Nodes: []NodeTemplateConfig{
							{
								Types:          []string{"Pod"},
								TemplateHashes: []string{"pod-v1"},
								LabelProperties: map[string]string{
									"name": "pod",
								},
							},
						},
						Relationships: []RelationshipTemplateConfig{
							{
								Type:         "OWNS",
								TemplateHash: "owns-v1",
								Source: RelationshipEndpointConfig{
									Type: "Namespace",
									MatchAttributes: SelectorAttributes{
										Labels: map[string]string{"name": "namespace"},
									},
								},
								Target: RelationshipEndpointConfig{
									Type: "Pod",
									MatchAttributes: SelectorAttributes{
										Labels: map[string]string{"name": "pod"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func intPointer(value int) *int {
	return &value
}
