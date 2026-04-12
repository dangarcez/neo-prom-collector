package engine

import (
	"testing"
	"time"

	"neo_collector_go/internal/config"
	"neo_collector_go/internal/domain"
)

func TestPlannerPlanBuildsNodesAndRelationships(t *testing.T) {
	planner := NewPlanner()

	job := config.JobConfig{
		Name:  "pods",
		Query: "kube_pod_info",
		Nodes: []config.NodeTemplateConfig{
			{
				Types:          []string{"Namespace"},
				TemplateHashes: []string{"namespace-v1"},
				LabelProperties: map[string]string{
					"name": "namespace",
				},
			},
			{
				Types:          []string{"Pod"},
				TemplateHashes: []string{"pod-v1"},
				LabelProperties: map[string]string{
					"name": "pod",
				},
			},
		},
		Relationships: []config.RelationshipTemplateConfig{
			{
				Type:         "OWNS",
				TemplateHash: "owns-v1",
				Source: config.RelationshipEndpointConfig{
					Type: "Namespace",
					MatchAttributes: config.SelectorAttributes{
						Labels: map[string]string{
							"name": "namespace",
						},
					},
				},
				Target: config.RelationshipEndpointConfig{
					Type: "Pod",
					MatchAttributes: config.SelectorAttributes{
						Labels: map[string]string{
							"name": "pod",
						},
					},
				},
			},
		},
	}
	job.Normalize(30)

	datapoint := domain.Datapoint{
		Labels: map[string]string{
			"namespace": "production",
			"pod":       "api-0",
		},
		Value:     1,
		Timestamp: time.Unix(1700000000, 0).UTC(),
	}

	plan, err := planner.Plan(job, datapoint)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(plan.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(plan.Nodes))
	}
	if len(plan.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(plan.Relationships))
	}

	foundNamespace := false
	foundPod := false
	for _, node := range plan.Nodes {
		if node.Properties["origin"] != "auto" {
			t.Fatalf("expected node origin to be auto, got: %#v", node.Properties["origin"])
		}
		if node.Properties["node_uid"] == "" {
			t.Fatal("expected node_uid to be set")
		}
		if node.Name == "production" {
			foundNamespace = true
		}
		if node.Name == "api-0" {
			foundPod = true
		}
	}

	if !foundNamespace || !foundPod {
		t.Fatalf("expected namespace and pod nodes, got: %#v", plan.Nodes)
	}

	relationship := plan.Relationships[0]
	if relationship.Properties["origin"] != "auto" {
		t.Fatalf("expected relationship origin to be auto, got: %#v", relationship.Properties["origin"])
	}
	if relationship.Properties["rel_uid"] == "" {
		t.Fatal("expected rel_uid to be set")
	}
	templateHashes, ok := relationship.Properties["template_hashes"].([]string)
	if !ok {
		t.Fatalf("expected template_hashes to be []string, got: %#v", relationship.Properties["template_hashes"])
	}
	if len(templateHashes) != 1 || templateHashes[0] != "owns-v1" {
		t.Fatalf("expected template_hashes to contain owns-v1, got: %#v", relationship.Properties["template_hashes"])
	}
}

func float64Pointer(value float64) *float64 {
	return &value
}
