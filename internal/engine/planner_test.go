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
		if node.Properties[domain.FieldOrigin] != "auto" {
			t.Fatalf("expected node origin to be auto, got: %#v", node.Properties[domain.FieldOrigin])
		}
		if node.Properties[domain.FieldNodeUID] == "" {
			t.Fatal("expected z4j_node_uid to be set")
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
	if relationship.Properties[domain.FieldOrigin] != "auto" {
		t.Fatalf("expected relationship origin to be auto, got: %#v", relationship.Properties[domain.FieldOrigin])
	}
	if relationship.Properties[domain.FieldRelUID] == "" {
		t.Fatal("expected z4j_rel_uid to be set")
	}
	if relationship.Properties[domain.FieldRelationshipTemplateHash] != "owns-v1" {
		t.Fatalf("expected z4j_template_hash to be owns-v1, got: %#v", relationship.Properties[domain.FieldRelationshipTemplateHash])
	}
	if _, ok := relationship.Properties["template_hash"]; ok {
		t.Fatalf("expected relationship not to contain template_hash, got: %#v", relationship.Properties["template_hash"])
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
	if node.Properties[domain.FieldNodeUID] != expectedNodeUID {
		t.Fatalf("expected node property z4j_node_uid %q, got %#v", expectedNodeUID, node.Properties[domain.FieldNodeUID])
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
	if relationship.Properties[domain.FieldRelUID] != expectedRelUID {
		t.Fatalf("expected relationship property z4j_rel_uid %q, got %#v", expectedRelUID, relationship.Properties[domain.FieldRelUID])
	}
}

func TestPlannerPlanAppliesPriorTransformToRelationshipSelectors(t *testing.T) {
	planner := NewPlanner()

	upperNameTransform := []config.PropertyTransformConfig{
		{
			Property: "name",
			Process: []config.PropertyProcessorConfig{
				{Type: config.PropertyProcessorTypeToUpper},
			},
		},
	}

	job := config.JobConfig{
		Name:  "pods",
		Query: "kube_pod_info",
		Nodes: []config.NodeTemplateConfig{
			{
				Types:              []string{"Namespace"},
				TemplateHashes:     []string{"namespace-v1"},
				LabelProperties:    map[string]string{"name": "namespace"},
				PropertyTransforms: upperNameTransform,
			},
			{
				Types:              []string{"Pod"},
				TemplateHashes:     []string{"pod-v1"},
				LabelProperties:    map[string]string{"name": "pod"},
				PropertyTransforms: upperNameTransform,
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
					PriorTransform: []config.PropertyTransformConfig{
						{
							Property: "namespace",
							Process: []config.PropertyProcessorConfig{
								{Type: config.PropertyProcessorTypeToUpper},
							},
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
					PriorTransform: []config.PropertyTransformConfig{
						{
							Property: "pod",
							Process: []config.PropertyProcessorConfig{
								{Type: config.PropertyProcessorTypeToUpper},
							},
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

	relationship := plan.Relationships[0]
	if relationship.Source.Attributes["name"] != "PRODUCTION" {
		t.Fatalf("expected transformed source selector name, got %#v", relationship.Source.Attributes["name"])
	}
	if relationship.Target.Attributes["name"] != "API-0" {
		t.Fatalf("expected transformed target selector name, got %#v", relationship.Target.Attributes["name"])
	}

	expectedRelUID := RelationshipUID(
		"OWNS",
		"owns-v1",
		domain.NodeSelector{
			Type: "Namespace",
			Attributes: map[string]any{
				"name": "PRODUCTION",
			},
		},
		domain.NodeSelector{
			Type: "Pod",
			Attributes: map[string]any{
				"name": "API-0",
			},
		},
	)
	if relationship.UID != expectedRelUID {
		t.Fatalf("expected relationship UID %q, got %q", expectedRelUID, relationship.UID)
	}
	if relationship.Properties[domain.FieldRelUID] != expectedRelUID {
		t.Fatalf("expected relationship property z4j_rel_uid %q, got %#v", expectedRelUID, relationship.Properties[domain.FieldRelUID])
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
	if _, exists := node.Properties[domain.FieldExpiresAt]; exists {
		t.Fatalf("expected planner to leave z4j_expires_at unset for nodes, got %#v", node.Properties[domain.FieldExpiresAt])
	}

	relationship := plan.Relationships[0]
	if relationship.ExpirationTimeMin == nil || *relationship.ExpirationTimeMin != 15 {
		t.Fatalf("expected relationship expiration metadata to be 15, got %#v", relationship.ExpirationTimeMin)
	}
	if _, exists := relationship.Properties[domain.FieldExpiresAt]; exists {
		t.Fatalf("expected planner to leave z4j_expires_at unset for relationships, got %#v", relationship.Properties[domain.FieldExpiresAt])
	}
}

func float64Pointer(value float64) *float64 {
	return &value
}
