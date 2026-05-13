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
                  - type: regex
                    pattern: "/(\\w+)_(\\w+)/"
                    output: "$1_and_$2"
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
	if len(transforms[0].Process) != 2 {
		t.Fatalf("expected two property processors, got %d", len(transforms[0].Process))
	}
	if transforms[0].Process[0].Type != PropertyProcessorTypeToUpper {
		t.Fatalf("expected property processor to normalize to %q, got %q", PropertyProcessorTypeToUpper, transforms[0].Process[0].Type)
	}
	if transforms[0].Process[1].Type != PropertyProcessorTypeRegex {
		t.Fatalf("expected regex property processor to normalize to %q, got %q", PropertyProcessorTypeRegex, transforms[0].Process[1].Type)
	}
	if transforms[0].Process[1].Pattern != `/(\w+)_(\w+)/` {
		t.Fatalf("expected regex pattern to be preserved, got %q", transforms[0].Process[1].Pattern)
	}
	if transforms[0].Process[1].Output != "$1_and_$2" {
		t.Fatalf("expected regex output to be preserved, got %q", transforms[0].Process[1].Output)
	}
}

func TestLoadFileNormalizesRelationshipEndpointPriorTransform(t *testing.T) {
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
        relationships:
          - type: OWNS
            template_hash: owns-v1
            source:
              type: Namespace
              match_attributes:
                labels:
                  name: namespace
              prior_transform:
                - property: namespace
                  process:
                    - type: to_upper
            target:
              type: Pod
              match_attributes:
                labels:
                  name: pod
              prior_transform:
                - property: pod
                  process:
                    - type: regex
                      pattern: "/(\\w+)-(\\d+)/"
                      output: "$1_$2"
`)

	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	sourceTransforms := cfg.PromTargets[0].Jobs[0].Relationships[0].Source.PriorTransform
	if len(sourceTransforms) != 1 {
		t.Fatalf("expected one source prior transform, got %d", len(sourceTransforms))
	}
	if sourceTransforms[0].Property != "namespace" {
		t.Fatalf("expected source prior transform property to be namespace, got %q", sourceTransforms[0].Property)
	}
	if sourceTransforms[0].Process[0].Type != PropertyProcessorTypeToUpper {
		t.Fatalf("expected source prior transform processor to normalize to %q, got %q", PropertyProcessorTypeToUpper, sourceTransforms[0].Process[0].Type)
	}

	targetTransforms := cfg.PromTargets[0].Jobs[0].Relationships[0].Target.PriorTransform
	if len(targetTransforms) != 1 {
		t.Fatalf("expected one target prior transform, got %d", len(targetTransforms))
	}
	if targetTransforms[0].Property != "pod" {
		t.Fatalf("expected target prior transform property to be pod, got %q", targetTransforms[0].Property)
	}
	if targetTransforms[0].Process[0].Type != PropertyProcessorTypeRegex {
		t.Fatalf("expected target prior transform processor to normalize to %q, got %q", PropertyProcessorTypeRegex, targetTransforms[0].Process[0].Type)
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

func TestLoadFileAcceptsAzureAuthWithoutManagedIdentityID(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
prom_targets:
  - name: azure-prom
    base_url: https://workspace.eastus.prometheus.monitor.azure.com
    azure_auth: {}
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

	if cfg.PromTargets[0].AzureAuth == nil {
		t.Fatal("expected azure_auth to be enabled")
	}
	if cfg.PromTargets[0].AzureAuth.ManagedIdentityID != "" {
		t.Fatalf("expected empty managed_identity_id, got %q", cfg.PromTargets[0].AzureAuth.ManagedIdentityID)
	}
}

func TestLoadFilePreservesAzureAuthManagedIdentityID(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
prom_targets:
  - name: azure-prom
    base_url: https://workspace.eastus.prometheus.monitor.azure.com
    azure_auth:
      managed_identity_id: "  /subscriptions/sub/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/prom-reader  "
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

	auth := cfg.PromTargets[0].AzureAuth
	if auth == nil {
		t.Fatal("expected azure_auth to be enabled")
	}

	want := "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/prom-reader"
	if auth.ManagedIdentityID != want {
		t.Fatalf("expected managed_identity_id %q, got %q", want, auth.ManagedIdentityID)
	}
}
