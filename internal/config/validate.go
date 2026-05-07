package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"neo_collector_go/internal/domain"
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

	if err := validateExpirationTimeMin(path+".expiration_time_min", node.ExpirationTimeMin); err != nil {
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

	if err := validatePropertyTransforms(path+".property_transforms", node.PropertyTransforms); err != nil {
		return err
	}

	return validateConditions(path+".conditions", node.Conditions)
}

func validateRelationshipTemplate(path string, relationship RelationshipTemplateConfig) error {
	if strings.TrimSpace(relationship.Type) == "" {
		return fmt.Errorf("%s.type is required", path)
	}

	if err := validateExpirationTimeMin(path+".expiration_time_min", relationship.ExpirationTimeMin); err != nil {
		return err
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

	if err := validatePropertyTransforms(path+".property_transforms", relationship.PropertyTransforms); err != nil {
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

	return validatePriorTransforms(path+".prior_transform", endpoint.PriorTransform)
}

func validateStaticProperties(path string, properties map[string]any) error {
	for key := range properties {
		if err := validateUserPropertyName(path, key); err != nil {
			return err
		}
	}

	return nil
}

func validateDynamicProperties(path string, properties map[string]string) error {
	for key, value := range properties {
		if err := validateUserPropertyName(path, key); err != nil {
			return err
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
		if err := validateUserPropertyName(itemPath+".name", property.Name); err != nil {
			return err
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

func validatePropertyTransforms(path string, transforms []PropertyTransformConfig) error {
	return validateTransformList(path, transforms, true)
}

func validatePriorTransforms(path string, transforms []PropertyTransformConfig) error {
	return validateTransformList(path, transforms, false)
}

func validateTransformList(path string, transforms []PropertyTransformConfig, requireUserPropertyName bool) error {
	for transformIndex, transform := range transforms {
		transformPath := fmt.Sprintf("%s[%d]", path, transformIndex)

		if strings.TrimSpace(transform.Property) == "" {
			return fmt.Errorf("%s.property is required", transformPath)
		}
		if requireUserPropertyName {
			if err := validateUserPropertyName(transformPath+".property", transform.Property); err != nil {
				return err
			}
		}

		if len(transform.Process) == 0 {
			return fmt.Errorf("%s.process must define at least one processor", transformPath)
		}

		for processorIndex, processor := range transform.Process {
			processorPath := fmt.Sprintf("%s.process[%d]", transformPath, processorIndex)
			if strings.TrimSpace(processor.Type) == "" {
				return fmt.Errorf("%s.type is required", processorPath)
			}
			if !IsSupportedPropertyProcessorType(processor.Type) {
				return fmt.Errorf("%s.type must be %s, %s or %s", processorPath, PropertyProcessorTypeToUpper, PropertyProcessorTypeToLower, PropertyProcessorTypeRegex)
			}
			if err := validatePropertyProcessor(processorPath, processor); err != nil {
				return err
			}
		}
	}

	return nil
}

func validatePropertyProcessor(path string, processor PropertyProcessorConfig) error {
	switch NormalizePropertyProcessorType(processor.Type) {
	case PropertyProcessorTypeRegex:
		return validateRegexPropertyProcessor(path, processor)
	default:
		return nil
	}
}

func validateRegexPropertyProcessor(path string, processor PropertyProcessorConfig) error {
	pattern := NormalizeRegexPattern(processor.Pattern)
	if pattern == "" {
		return fmt.Errorf("%s.pattern is required for %s processors", path, PropertyProcessorTypeRegex)
	}

	if strings.TrimSpace(processor.Output) == "" {
		return fmt.Errorf("%s.output is required for %s processors", path, PropertyProcessorTypeRegex)
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("%s.pattern is invalid: %w", path, err)
	}

	groupCount := compiled.NumSubexp()
	if groupCount == 0 {
		return fmt.Errorf("%s.pattern must define at least one capture group", path)
	}

	references := regexOutputGroupReferences(processor.Output)
	if len(references) == 0 {
		return fmt.Errorf("%s.output must reference at least one capture group", path)
	}
	for _, reference := range references {
		if reference == 0 {
			return fmt.Errorf("%s.output references $0, but regex output groups start at $1", path)
		}
		if reference > groupCount {
			return fmt.Errorf("%s.output references $%d, but pattern defines only %d capture group(s)", path, reference, groupCount)
		}
	}

	return nil
}

func regexOutputGroupReferences(output string) []int {
	matches := regexp.MustCompile(`\$(\d+)`).FindAllStringSubmatch(output, -1)
	references := make([]int, 0, len(matches))
	for _, match := range matches {
		reference, err := strconv.Atoi(match[1])
		if err == nil {
			references = append(references, reference)
		}
	}
	return references
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

func validateExpirationTimeMin(path string, value *int) error {
	if value == nil {
		return nil
	}

	if *value <= 0 {
		return fmt.Errorf("%s must be greater than zero when provided", path)
	}

	return nil
}

func validateUserPropertyName(path string, name string) error {
	propertyName := strings.TrimSpace(name)
	if propertyName == "" {
		return fmt.Errorf("%s contains an empty property name", path)
	}
	if strings.HasPrefix(propertyName, domain.AppFieldPrefix) {
		return fmt.Errorf("%s property %q uses reserved prefix %q", path, name, domain.AppFieldPrefix)
	}
	return nil
}
