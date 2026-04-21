package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileAcceptsFractionalSleepSeconds(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
prom_targets:
  - name: prom
    base_url: http://localhost:9090
    runtime:
      default_interval_seconds: 30
      sleep_seconds: 0.25
      dry_run: true
    jobs:
      - name: job
        query: up
        nodes:
          - type: Pod
            template_hashes:
              - pod-v1
            label_properties:
              name: pod
`)

	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	if cfg.PromTargets[0].Runtime.SleepSeconds != 0.25 {
		t.Fatalf("expected fractional sleep_seconds to be preserved, got %v", cfg.PromTargets[0].Runtime.SleepSeconds)
	}
}

func TestLoadFileNormalizesPropertyTransforms(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
prom_targets:
  - name: prom
    base_url: http://localhost:9090
    runtime:
      default_interval_seconds: 30
      dry_run: true
    jobs:
      - name: job
        query: up
        nodes:
          - type: Pod
            template_hashes:
              - pod-v1
            label_properties:
              name: pod
            property_transforms:
              - property: name
                process:
                  - type: to_upper
`)

	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	transforms := cfg.PromTargets[0].Jobs[0].Nodes[0].PropertyTransforms
	if len(transforms) != 1 {
		t.Fatalf("expected one property transform, got %d", len(transforms))
	}
	if transforms[0].Property != "name" {
		t.Fatalf("expected transform property to be name, got %q", transforms[0].Property)
	}
	if len(transforms[0].Process) != 1 {
		t.Fatalf("expected one property processor, got %d", len(transforms[0].Process))
	}
	if transforms[0].Process[0].Type != PropertyProcessorTypeToUpper {
		t.Fatalf("expected property processor to normalize to %q, got %q", PropertyProcessorTypeToUpper, transforms[0].Process[0].Type)
	}
}

func TestLoadFilePreservesExpirationTimeMin(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
prom_targets:
  - name: prom
    base_url: http://localhost:9090
    runtime:
      default_interval_seconds: 30
      dry_run: true
    jobs:
      - name: job
        query: up
        nodes:
          - type: Pod
            template_hashes:
              - pod-v1
            expiration_time_min: 30
            label_properties:
              name: pod
        relationships:
          - type: OWNS
            template_hash: owns-v1
            expiration_time_min: 15
            source:
              type: Namespace
              match_attributes:
                labels:
                  name: namespace
            target:
              type: Pod
              match_attributes:
                labels:
                  name: pod
`)

	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	nodeExpiration := cfg.PromTargets[0].Jobs[0].Nodes[0].ExpirationTimeMin
	if nodeExpiration == nil || *nodeExpiration != 30 {
		t.Fatalf("expected node expiration_time_min to be 30, got %#v", nodeExpiration)
	}

	relationshipExpiration := cfg.PromTargets[0].Jobs[0].Relationships[0].ExpirationTimeMin
	if relationshipExpiration == nil || *relationshipExpiration != 15 {
		t.Fatalf("expected relationship expiration_time_min to be 15, got %#v", relationshipExpiration)
	}
}
