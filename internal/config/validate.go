package config

import (
	"fmt"
	"strings"
)

func ValidateFileConfig(cfg FileConfig) error {
	if len(cfg.PromTargets) == 0 {
		return fmt.Errorf("config must define at least one prom target")
	}

	for targetIndex, target := range cfg.PromTargets {
		targetPath := fmt.Sprintf("prom_targets[%d]", targetIndex)

		if strings.TrimSpace(target.Name) == "" {
			return fmt.Errorf("%s.name is required", targetPath)
		}

		if strings.TrimSpace(target.BaseURL) == "" {
			return fmt.Errorf("%s.base_url is required", targetPath)
		}

		if len(target.Jobs) == 0 {
			return fmt.Errorf("%s.jobs must define at least one job", targetPath)
		}

		for jobIndex, job := range target.Jobs {
			jobPath := fmt.Sprintf("%s.jobs[%d]", targetPath, jobIndex)

			if strings.TrimSpace(job.Name) == "" {
				return fmt.Errorf("%s.name is required", jobPath)
			}

			if strings.TrimSpace(job.Query) == "" {
				return fmt.Errorf("%s.query is required", jobPath)
			}

			if job.IntervalSeconds <= 0 {
				return fmt.Errorf("%s.interval_seconds must be greater than zero", jobPath)
			}

			for nodeIndex, node := range job.Nodes {
				if err := validateNodeTemplate(fmt.Sprintf("%s.nodes[%d]", jobPath, nodeIndex), node); err != nil {
					return err
				}
			}

			for relIndex, relationship := range job.Relationships {
				if err := validateRelationshipTemplate(fmt.Sprintf("%s.relationships[%d]", jobPath, relIndex), relationship); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func validateNodeTemplate(path string, node NodeTemplateConfig) error {
	if err := validateUpdatePolicy(path+".update_policy", node.UpdatePolicy); err != nil {
		return err
	}

	if len(node.NormalizedTypes()) == 0 {
		return fmt.Errorf("%s must define at least one type", path)
	}

	if len(node.TemplateHashes) == 0 {
		return fmt.Errorf("%s.template_hashes must define at least one value", path)
	}

	if !node.HasNameProperty() {
		return fmt.Errorf("%s must define the name property in static_properties or label_properties", path)
	}

	if err := validateStaticProperties(path+".static_properties", node.StaticProperties); err != nil {
		return err
	}

	if err := validateDynamicProperties(path+".label_properties", node.LabelProperties); err != nil {
		return err
	}

	if err := validateConditionalProperties(path+".conditional_properties", node.ConditionalProperties); err != nil {
		return err
	}

	return validateConditions(path+".conditions", node.Conditions)
}

func validateRelationshipTemplate(path string, relationship RelationshipTemplateConfig) error {
	if strings.TrimSpace(relationship.Type) == "" {
		return fmt.Errorf("%s.type is required", path)
	}

	if err := validateUpdatePolicy(path+".update_policy", relationship.UpdatePolicy); err != nil {
		return err
	}

	if strings.TrimSpace(relationship.NormalizedTemplateHash()) == "" {
		return fmt.Errorf("%s.template_hash is required", path)
	}

	if err := validateStaticProperties(path+".static_properties", relationship.StaticProperties); err != nil {
		return err
	}

	if err := validateDynamicProperties(path+".label_properties", relationship.LabelProperties); err != nil {
		return err
	}

	if err := validateConditionalProperties(path+".conditional_properties", relationship.ConditionalProperties); err != nil {
		return err
	}

	if err := validateConditions(path+".conditions", relationship.Conditions); err != nil {
		return err
	}

	if err := validateSelector(path+".source", relationship.Source); err != nil {
		return err
	}

	return validateSelector(path+".target", relationship.Target)
}

func validateSelector(path string, endpoint RelationshipEndpointConfig) error {
	if strings.TrimSpace(endpoint.Type) == "" {
		return fmt.Errorf("%s.type is required", path)
	}

	if len(endpoint.MatchAttributes.Static) == 0 && len(endpoint.MatchAttributes.Labels) == 0 {
		return fmt.Errorf("%s must define at least one match attribute", path)
	}

	for key := range endpoint.MatchAttributes.Static {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("%s contains an empty static attribute name", path)
		}
	}

	for key, value := range endpoint.MatchAttributes.Labels {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("%s contains an empty dynamic attribute name", path)
		}

		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s dynamic attribute %q has an empty source token", path, key)
		}
	}

	return nil
}

func validateStaticProperties(path string, properties map[string]any) error {
	for key := range properties {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("%s contains an empty property name", path)
		}
	}

	return nil
}

func validateDynamicProperties(path string, properties map[string]string) error {
	for key, value := range properties {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("%s contains an empty property name", path)
		}

		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s property %q has an empty source token", path, key)
		}
	}

	return nil
}

func validateConditionalProperties(path string, properties []ConditionalPropertyConfig) error {
	for index, property := range properties {
		itemPath := fmt.Sprintf("%s[%d]", path, index)

		if strings.TrimSpace(property.Name) == "" {
			return fmt.Errorf("%s.name is required", itemPath)
		}

		switch strings.ToLower(strings.TrimSpace(property.Type)) {
		case "static":
			if property.Value == nil {
				return fmt.Errorf("%s.value is required for static conditional properties", itemPath)
			}
		case "label":
			if strings.TrimSpace(property.FromLabel) == "" {
				return fmt.Errorf("%s.from_label is required for label conditional properties", itemPath)
			}
		default:
			return fmt.Errorf("%s.type must be static or label", itemPath)
		}

		if len(property.Conditions) == 0 {
			return fmt.Errorf("%s.conditions must define at least one condition", itemPath)
		}

		if err := validateConditions(itemPath+".conditions", property.Conditions); err != nil {
			return err
		}
	}

	return nil
}

func validateConditions(path string, conditions []ConditionConfig) error {
	for index, condition := range conditions {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		if err := validateCondition(itemPath, condition); err != nil {
			return err
		}
	}

	return nil
}

func validateCondition(path string, condition ConditionConfig) error {
	switch strings.ToLower(strings.TrimSpace(condition.Type)) {
	case "label":
		if strings.TrimSpace(condition.Label) == "" {
			return fmt.Errorf("%s.label is required for label conditions", path)
		}

		operatorCount := 0
		if condition.Equals != nil {
			operatorCount++
		}
		if condition.NotEquals != nil {
			operatorCount++
		}
		if condition.GreaterThan != nil || condition.LessThan != nil {
			return fmt.Errorf("%s label conditions only support equals and not_equals", path)
		}
		if operatorCount != 1 {
			return fmt.Errorf("%s label conditions require exactly one operator", path)
		}
	case "label_exists":
		if strings.TrimSpace(condition.Label) == "" {
			return fmt.Errorf("%s.label is required for label_exists conditions", path)
		}

		operatorCount := 0
		if condition.Equals != nil {
			operatorCount++
		}
		if condition.NotEquals != nil {
			operatorCount++
		}
		if condition.GreaterThan != nil {
			operatorCount++
		}
		if condition.LessThan != nil {
			operatorCount++
		}
		if operatorCount != 0 {
			return fmt.Errorf("%s label_exists conditions do not accept operators", path)
		}
	case "value":
		operatorCount := 0
		if condition.Equals != nil {
			operatorCount++
		}
		if condition.NotEquals != nil {
			operatorCount++
		}
		if condition.GreaterThan != nil {
			operatorCount++
		}
		if condition.LessThan != nil {
			operatorCount++
		}
		if operatorCount != 1 {
			return fmt.Errorf("%s value conditions require exactly one operator", path)
		}
	default:
		return fmt.Errorf("%s.type must be label, label_exists or value", path)
	}

	return nil
}

func validateUpdatePolicy(path, value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "create", "merge", "merge_at_change", "mergeatchange", "merge-at-change":
		return nil
	default:
		return fmt.Errorf("%s must be create, merge or merge_at_change", path)
	}
}
