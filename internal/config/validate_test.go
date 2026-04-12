package config

import (
	"strings"
	"testing"
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
