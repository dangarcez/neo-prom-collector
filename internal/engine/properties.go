package engine

import (
	"fmt"
	"strings"
	"time"

	"neo_collector_go/internal/config"
	"neo_collector_go/internal/domain"
)

func ResolveProperties(
	staticProperties map[string]any,
	dynamicProperties map[string]string,
	conditionalProperties []config.ConditionalPropertyConfig,
	datapoint domain.Datapoint,
) (map[string]any, error) {
	properties := cloneMap(staticProperties)

	for propertyName, sourceToken := range dynamicProperties {
		resolved, found, err := ResolveOptionalToken(sourceToken, datapoint)
		if err != nil {
			return nil, fmt.Errorf("resolve dynamic property %q: %w", propertyName, err)
		}
		if !found {
			continue
		}
		properties[propertyName] = resolved
	}

	for _, property := range conditionalProperties {
		matches, err := MatchConditions(property.Conditions, datapoint)
		if err != nil {
			return nil, fmt.Errorf("evaluate conditional property %q: %w", property.Name, err)
		}
		if !matches {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(property.Type)) {
		case "static":
			properties[property.Name] = property.Value
		case "label":
			resolved, found, err := ResolveOptionalToken(property.FromLabel, datapoint)
			if err != nil {
				return nil, fmt.Errorf("resolve conditional property %q: %w", property.Name, err)
			}
			if !found {
				continue
			}
			properties[property.Name] = resolved
		default:
			return nil, fmt.Errorf("unsupported conditional property type %q", property.Type)
		}
	}

	return properties, nil
}

func ResolveSelector(endpoint config.RelationshipEndpointConfig, datapoint domain.Datapoint) (domain.NodeSelector, error) {
	attributes := cloneMap(endpoint.MatchAttributes.Static)

	for attributeName, sourceToken := range endpoint.MatchAttributes.Labels {
		resolved, err := ResolveToken(sourceToken, datapoint)
		if err != nil {
			return domain.NodeSelector{}, fmt.Errorf("resolve selector attribute %q: %w", attributeName, err)
		}
		attributes[attributeName] = resolved
	}

	return domain.NodeSelector{
		Type:       endpoint.Type,
		Attributes: attributes,
	}, nil
}

func ResolveOptionalToken(sourceToken string, datapoint domain.Datapoint) (any, bool, error) {
	switch sourceToken {
	case "__value__":
		return datapoint.Value, true, nil
	case "__timestamp__":
		return datapoint.Timestamp.UTC().Format(time.RFC3339), true, nil
	default:
		value, ok := datapoint.Labels[sourceToken]
		if !ok {
			return nil, false, nil
		}
		return value, true, nil
	}
}

func ResolveToken(sourceToken string, datapoint domain.Datapoint) (any, error) {
	value, found, err := ResolveOptionalToken(sourceToken, datapoint)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("label %q was not found in datapoint", sourceToken)
	}
	return value, nil
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}

	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
