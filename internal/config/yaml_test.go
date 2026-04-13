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
