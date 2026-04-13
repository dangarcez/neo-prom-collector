package engine

import (
	"fmt"
	"strconv"
	"strings"

	"neo_collector_go/internal/config"
	"neo_collector_go/internal/domain"
)

func MatchConditions(conditions []config.ConditionConfig, datapoint domain.Datapoint) (bool, error) {
	for _, condition := range conditions {
		matches, err := matchCondition(condition, datapoint)
		if err != nil {
			return false, err
		}
		if !matches {
			return false, nil
		}
	}

	return true, nil
}

func matchCondition(condition config.ConditionConfig, datapoint domain.Datapoint) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(condition.Type)) {
	case "label":
		actualValue, ok := datapoint.Labels[condition.Label]
		if !ok {
			return false, nil
		}

		if condition.Equals != nil {
			return actualValue == fmt.Sprint(condition.Equals), nil
		}

		if condition.NotEquals != nil {
			return actualValue != fmt.Sprint(condition.NotEquals), nil
		}

		return false, fmt.Errorf("unsupported label condition")
	case "label_exists":
		_, ok := datapoint.Labels[condition.Label]
		return ok, nil
	case "value":
		if condition.Equals != nil {
			expected, err := toFloat64(condition.Equals)
			if err != nil {
				return false, err
			}
			return datapoint.Value == expected, nil
		}

		if condition.NotEquals != nil {
			expected, err := toFloat64(condition.NotEquals)
			if err != nil {
				return false, err
			}
			return datapoint.Value != expected, nil
		}

		if condition.GreaterThan != nil {
			return datapoint.Value > *condition.GreaterThan, nil
		}

		if condition.LessThan != nil {
			return datapoint.Value < *condition.LessThan, nil
		}

		return false, fmt.Errorf("unsupported value condition")
	default:
		return false, fmt.Errorf("unsupported condition type %q", condition.Type)
	}
}

func toFloat64(value any) (float64, error) {
	switch typed := value.(type) {
	case int:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case float32:
		return float64(typed), nil
	case float64:
		return typed, nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, fmt.Errorf("expected numeric value, got %q", typed)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", value)
	}
}
