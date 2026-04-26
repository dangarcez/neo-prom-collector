package engine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"neo_collector_go/internal/config"
)

type propertyProcessor interface {
	Apply(value any, process config.PropertyProcessorConfig) (any, bool, error)
}

type propertyProcessorFunc func(value any, process config.PropertyProcessorConfig) (any, bool, error)

func (f propertyProcessorFunc) Apply(value any, process config.PropertyProcessorConfig) (any, bool, error) {
	return f(value, process)
}

var propertyProcessorRegistry = map[string]propertyProcessor{
	config.PropertyProcessorTypeToUpper: newStringPropertyProcessor(strings.ToUpper),
	config.PropertyProcessorTypeToLower: newStringPropertyProcessor(strings.ToLower),
	config.PropertyProcessorTypeRegex:   regexPropertyProcessor{},
}

func ApplyPropertyTransforms(properties map[string]any, transforms []config.PropertyTransformConfig) error {
	for _, transform := range transforms {
		currentValue, exists := properties[transform.Property]
		if !exists {
			continue
		}

		for _, process := range transform.Process {
			processorType := config.NormalizePropertyProcessorType(process.Type)
			processor, ok := propertyProcessorRegistry[processorType]
			if !ok {
				return fmt.Errorf("unsupported property processor type %q for property %q", process.Type, transform.Property)
			}

			nextValue, applied, err := processor.Apply(currentValue, process)
			if err != nil {
				return fmt.Errorf("apply processor %q to property %q: %w", processorType, transform.Property, err)
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
	return propertyProcessorFunc(func(value any, _ config.PropertyProcessorConfig) (any, bool, error) {
		text, ok := value.(string)
		if !ok {
			return value, false, nil
		}

		return transform(text), true, nil
	})
}

type regexPropertyProcessor struct{}

func (regexPropertyProcessor) Apply(value any, process config.PropertyProcessorConfig) (any, bool, error) {
	text, ok := value.(string)
	if !ok {
		return value, false, nil
	}

	pattern := config.NormalizeRegexPattern(process.Pattern)
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return value, false, err
	}

	matches := compiled.FindStringSubmatch(text)
	if matches == nil {
		return value, false, nil
	}

	return expandRegexOutput(process.Output, matches), true, nil
}

func expandRegexOutput(output string, matches []string) string {
	return regexp.MustCompile(`\$(\d+)`).ReplaceAllStringFunc(output, func(token string) string {
		index, err := strconv.Atoi(strings.TrimPrefix(token, "$"))
		if err != nil || index <= 0 || index >= len(matches) {
			return ""
		}
		return matches[index]
	})
}
