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
	if relationship.Properties["template_hash"] != "owns-v1" {
		t.Fatalf("expected template_hash to be owns-v1, got: %#v", relationship.Properties["template_hash"])
	}
	if _, ok := relationship.Properties["template_hashes"]; ok {
		t.Fatalf("expected relationship not to contain template_hashes, got: %#v", relationship.Properties["template_hashes"])
	}
}

func TestPlannerPlanSkipsMissingOptionalLabelProperties(t *testing.T) {
	planner := NewPlanner()

	job := config.JobConfig{
		Name:  "pods",
		Query: "kube_pod_info",
		Nodes: []config.NodeTemplateConfig{
			{
				Types:          []string{"Pod"},
				TemplateHashes: []string{"pod-v1"},
				LabelProperties: map[string]string{
					"name":      "pod",
					"namespace": "namespace",
				},
			},
		},
		Relationships: []config.RelationshipTemplateConfig{
			{
				Type:         "BELONGS_TO",
				TemplateHash: "pod-belongs-to-cluster-v1",
				LabelProperties: map[string]string{
					"missing_label": "cluster",
				},
				Source: config.RelationshipEndpointConfig{
					Type: "Pod",
					MatchAttributes: config.SelectorAttributes{
						Labels: map[string]string{
							"name": "pod",
						},
					},
				},
				Target: config.RelationshipEndpointConfig{
					Type: "Cluster",
					MatchAttributes: config.SelectorAttributes{
						Static: map[string]any{
							"name": "main",
						},
					},
				},
			},
		},
	}
	job.Normalize(30)

	datapoint := domain.Datapoint{
		Labels: map[string]string{
			"pod": "api-0",
		},
		Value:     1,
		Timestamp: time.Unix(1700000000, 0).UTC(),
	}

	plan, err := planner.Plan(job, datapoint)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(plan.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(plan.Nodes))
	}
	if len(plan.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(plan.Relationships))
	}

	if _, exists := plan.Nodes[0].Properties["namespace"]; exists {
		t.Fatalf("expected missing node label property to be omitted, got: %#v", plan.Nodes[0].Properties["namespace"])
	}
	if _, exists := plan.Relationships[0].Properties["missing_label"]; exists {
		t.Fatalf("expected missing relationship label property to be omitted, got: %#v", plan.Relationships[0].Properties["missing_label"])
	}
}

func TestPlannerPlanAppliesPropertyTransforms(t *testing.T) {
	planner := NewPlanner()

	job := config.JobConfig{
		Name:  "pods",
		Query: "kube_pod_info",
		Nodes: []config.NodeTemplateConfig{
			{
				Types:          []string{"Pod"},
				TemplateHashes: []string{"pod-v1"},
				LabelProperties: map[string]string{
					"name": "pod",
				},
				PropertyTransforms: []config.PropertyTransformConfig{
					{
						Property: "name",
						Process: []config.PropertyProcessorConfig{
							{Type: config.PropertyProcessorTypeToUpper},
						},
					},
				},
			},
		},
		Relationships: []config.RelationshipTemplateConfig{
			{
				Type:         "OWNS",
				TemplateHash: "owns-v1",
				LabelProperties: map[string]string{
					"environment": "environment",
				},
				PropertyTransforms: []config.PropertyTransformConfig{
					{
						Property: "environment",
						Process: []config.PropertyProcessorConfig{
							{Type: config.PropertyProcessorTypeToUpper},
						},
					},
				},
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
			"namespace":   "production",
			"pod":         "api-0",
			"environment": "prod",
		},
		Value:     1,
		Timestamp: time.Unix(1700000000, 0).UTC(),
	}

	plan, err := planner.Plan(job, datapoint)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(plan.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(plan.Nodes))
	}
	if len(plan.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(plan.Relationships))
	}

	node := plan.Nodes[0]
	if node.Name != "API-0" {
		t.Fatalf("expected transformed node name, got %q", node.Name)
	}
	if node.Properties["name"] != "API-0" {
		t.Fatalf("expected transformed node property name, got %#v", node.Properties["name"])
	}

	expectedNodeUID := NodeUID([]string{"Pod"}, "API-0", []string{"pod-v1"})
	if node.UID != expectedNodeUID {
		t.Fatalf("expected node UID %q, got %q", expectedNodeUID, node.UID)
	}
	if node.Properties["node_uid"] != expectedNodeUID {
		t.Fatalf("expected node property node_uid %q, got %#v", expectedNodeUID, node.Properties["node_uid"])
	}

	relationship := plan.Relationships[0]
	if relationship.Properties["environment"] != "PROD" {
		t.Fatalf("expected transformed relationship property, got %#v", relationship.Properties["environment"])
	}

	expectedRelUID := RelationshipUID(
		"OWNS",
		"owns-v1",
		domain.NodeSelector{
			Type: "Namespace",
			Attributes: map[string]any{
				"name": "production",
			},
		},
		domain.NodeSelector{
			Type: "Pod",
			Attributes: map[string]any{
				"name": "api-0",
			},
		},
	)
	if relationship.UID != expectedRelUID {
		t.Fatalf("expected relationship UID %q, got %q", expectedRelUID, relationship.UID)
	}
	if relationship.Properties["rel_uid"] != expectedRelUID {
		t.Fatalf("expected relationship property rel_uid %q, got %#v", expectedRelUID, relationship.Properties["rel_uid"])
	}
}

func TestPlannerPlanCarriesExpirationMetadataWithoutInjectingExpiresAt(t *testing.T) {
	planner := NewPlanner()
	nodeExpiration := 30
	relationshipExpiration := 15

	job := config.JobConfig{
		Name:  "pods",
		Query: "kube_pod_info",
		Nodes: []config.NodeTemplateConfig{
			{
				Types:             []string{"Pod"},
				TemplateHashes:    []string{"pod-v1"},
				ExpirationTimeMin: &nodeExpiration,
				LabelProperties: map[string]string{
					"name": "pod",
				},
			},
		},
		Relationships: []config.RelationshipTemplateConfig{
			{
				Type:              "OWNS",
				TemplateHash:      "owns-v1",
				ExpirationTimeMin: &relationshipExpiration,
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

	node := plan.Nodes[0]
	if node.ExpirationTimeMin == nil || *node.ExpirationTimeMin != 30 {
		t.Fatalf("expected node expiration metadata to be 30, got %#v", node.ExpirationTimeMin)
	}
	if _, exists := node.Properties["expires_at"]; exists {
		t.Fatalf("expected planner to leave expires_at unset for nodes, got %#v", node.Properties["expires_at"])
	}

	relationship := plan.Relationships[0]
	if relationship.ExpirationTimeMin == nil || *relationship.ExpirationTimeMin != 15 {
		t.Fatalf("expected relationship expiration metadata to be 15, got %#v", relationship.ExpirationTimeMin)
	}
	if _, exists := relationship.Properties["expires_at"]; exists {
		t.Fatalf("expected planner to leave expires_at unset for relationships, got %#v", relationship.Properties["expires_at"])
	}
}

func float64Pointer(value float64) *float64 {
	return &value
}
