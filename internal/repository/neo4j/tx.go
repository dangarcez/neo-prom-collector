package neo4j

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	driver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"neo_collector_go/internal/domain"
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func sanitizeIdentifier(value string) (string, error) {
	if !identifierPattern.MatchString(value) {
		return "", fmt.Errorf("invalid cypher identifier %q", value)
	}

	return value, nil
}

func labelsFragment(labels []string) (string, error) {
	sanitized := []string{"Entity"}
	seen := map[string]struct{}{
		"Entity": {},
	}

	for _, label := range labels {
		identifier, err := sanitizeIdentifier(label)
		if err != nil {
			return "", err
		}
		if _, exists := seen[identifier]; exists {
			continue
		}
		seen[identifier] = struct{}{}
		sanitized = append(sanitized, identifier)
	}

	return ":" + strings.Join(sanitized, ":"), nil
}

func buildPropertyFilters(alias string, properties map[string]any, prefix string) (string, map[string]any, error) {
	if len(properties) == 0 {
		return "", map[string]any{}, nil
	}

	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	params := make(map[string]any, len(keys))

	for _, key := range keys {
		parameterName := sanitizeParameterName(fmt.Sprintf("%s_%s", prefix, key))
		parts = append(parts, fmt.Sprintf("%s.%s = $%s", alias, cypherProperty(key), parameterName))
		params[parameterName] = properties[key]
	}

	return strings.Join(parts, " AND "), params, nil
}

func cypherProperty(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func sanitizeParameterName(value string) string {
	replacer := strings.NewReplacer(".", "_", "-", "_", " ", "_")
	return replacer.Replace(value)
}

func executeActionQuery(ctx context.Context, tx driver.ManagedTransaction, query string, params map[string]any) (domain.PersistAction, error) {
	result, err := tx.Run(ctx, query, params)
	if err != nil {
		return "", fmt.Errorf("run action query: %w", err)
	}

	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("action query returned no rows")
	}

	value, ok := result.Record().Get("action")
	if !ok {
		return "", fmt.Errorf("action query did not return action")
	}

	action, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("action has unexpected type %T", value)
	}

	if err := result.Err(); err != nil {
		return "", err
	}

	return domain.PersistAction(action), nil
}

func executeCountQuery(ctx context.Context, tx driver.ManagedTransaction, query string, params map[string]any, field string) (int, error) {
	result, err := tx.Run(ctx, query, params)
	if err != nil {
		return 0, fmt.Errorf("run count query: %w", err)
	}

	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("count query returned no rows")
	}

	value, ok := result.Record().Get(field)
	if !ok {
		return 0, fmt.Errorf("count query did not return %s", field)
	}

	count, ok := value.(int64)
	if !ok {
		return 0, fmt.Errorf("count has unexpected type %T", value)
	}

	if err := result.Err(); err != nil {
		return 0, err
	}

	return int(count), nil
}

func cloneMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
