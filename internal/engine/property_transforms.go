package engine

import (
	"fmt"
	"strings"

	"neo_collector_go/internal/config"
)

type propertyProcessor interface {
	Apply(value any) (any, bool, error)
}

type propertyProcessorFunc func(value any) (any, bool, error)

func (f propertyProcessorFunc) Apply(value any) (any, bool, error) {
	return f(value)
}

var propertyProcessorRegistry = map[string]propertyProcessor{
	config.PropertyProcessorTypeToUpper: newStringPropertyProcessor(strings.ToUpper),
	config.PropertyProcessorTypeToLower: newStringPropertyProcessor(strings.ToLower),
}

func ApplyPropertyTransforms(properties map[string]any, transforms []config.PropertyTransformConfig) error {
	for _, transform := range transforms {
		currentValue, exists := properties[transform.Property]
		if !exists {
			continue
		}

		for _, process := range transform.Process {
			processor, ok := propertyProcessorRegistry[process.Type]
			if !ok {
				return fmt.Errorf("unsupported property processor type %q for property %q", process.Type, transform.Property)
			}

			nextValue, applied, err := processor.Apply(currentValue)
			if err != nil {
				return fmt.Errorf("apply processor %q to property %q: %w", process.Type, transform.Property, err)
			}
			if !applied {
				continue
			}

			currentValue = nextValue
		}

		properties[transform.Property] = currentValue
	}

	return nil
}

func newStringPropertyProcessor(transform func(string) string) propertyProcessor {
	return propertyProcessorFunc(func(value any) (any, bool, error) {
		text, ok := value.(string)
		if !ok {
			return value, false, nil
		}

		return transform(text), true, nil
	})
}
