package engine

import (
	"fmt"
	"sort"
	"strings"

	"neo_collector_go/internal/config"
	"neo_collector_go/internal/domain"
)

type Planner struct{}

func NewPlanner() *Planner {
	return &Planner{}
}

func (p *Planner) Plan(job config.JobConfig, datapoint domain.Datapoint) (domain.MutationPlan, error) {
	plan := domain.MutationPlan{
		Nodes:         []domain.GraphNode{},
		Relationships: []domain.GraphRelationship{},
	}

	for _, nodeTemplate := range job.Nodes {
		matches, err := MatchConditions(nodeTemplate.Conditions, datapoint)
		if err != nil {
			return domain.MutationPlan{}, fmt.Errorf("evaluate node conditions: %w", err)
		}
		if !matches {
			continue
		}

		properties, err := ResolveProperties(
			nodeTemplate.StaticProperties,
			nodeTemplate.LabelProperties,
			nodeTemplate.ConditionalProperties,
			nodeTemplate.PropertyTransforms,
			datapoint,
		)
		if err != nil {
			return domain.MutationPlan{}, fmt.Errorf("resolve node properties: %w", err)
		}

		nameValue, ok := properties["name"]
		if !ok {
			return domain.MutationPlan{}, fmt.Errorf("resolved node properties do not contain name")
		}

		name := strings.TrimSpace(fmt.Sprint(nameValue))
		if name == "" || name == "<nil>" {
			return domain.MutationPlan{}, fmt.Errorf("resolved node name is empty")
		}

		types := nodeTemplate.NormalizedTypes()
		sort.Strings(types)

		uid := NodeUID(types, name, nodeTemplate.TemplateHashes)
		properties["name"] = name
		properties["node_uid"] = uid
		properties["template_hashes"] = append([]string(nil), nodeTemplate.TemplateHashes...)
		properties["origin"] = "auto"

		plan.Nodes = append(plan.Nodes, domain.GraphNode{
			Types:             types,
			Name:              name,
			TemplateHashes:    append([]string(nil), nodeTemplate.TemplateHashes...),
			UpdatePolicy:      domain.NormalizeUpdatePolicy(nodeTemplate.UpdatePolicy),
			ExpirationTimeMin: cloneIntPointer(nodeTemplate.ExpirationTimeMin),
			Properties:        properties,
			UID:               uid,
		})
	}

	for _, relationshipTemplate := range job.Relationships {
		matches, err := MatchConditions(relationshipTemplate.Conditions, datapoint)
		if err != nil {
			return domain.MutationPlan{}, fmt.Errorf("evaluate relationship conditions: %w", err)
		}
		if !matches {
			continue
		}

		properties, err := ResolveProperties(
			relationshipTemplate.StaticProperties,
			relationshipTemplate.LabelProperties,
			relationshipTemplate.ConditionalProperties,
			relationshipTemplate.PropertyTransforms,
			datapoint,
		)
		if err != nil {
			return domain.MutationPlan{}, fmt.Errorf("resolve relationship properties: %w", err)
		}

		source, err := ResolveSelector(relationshipTemplate.Source, datapoint)
		if err != nil {
			return domain.MutationPlan{}, fmt.Errorf("resolve relationship source: %w", err)
		}

		target, err := ResolveSelector(relationshipTemplate.Target, datapoint)
		if err != nil {
			return domain.MutationPlan{}, fmt.Errorf("resolve relationship target: %w", err)
		}

		templateHash := relationshipTemplate.NormalizedTemplateHash()
		uid := RelationshipUID(relationshipTemplate.Type, templateHash, source, target)

		properties["rel_uid"] = uid
		properties["template_hashes"] = []string{templateHash}
		properties["origin"] = "auto"

		plan.Relationships = append(plan.Relationships, domain.GraphRelationship{
			Type:              relationshipTemplate.Type,
			TemplateHash:      templateHash,
			UpdatePolicy:      domain.NormalizeUpdatePolicy(relationshipTemplate.UpdatePolicy),
			ExpirationTimeMin: cloneIntPointer(relationshipTemplate.ExpirationTimeMin),
			Source:            source,
			Target:            target,
			Properties:        properties,
			UID:               uid,
		})
	}

	return plan, nil
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}
