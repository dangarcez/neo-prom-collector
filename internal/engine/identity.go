package engine

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"neo_collector_go/internal/domain"
)

func NodeUID(types []string, name string, templateHashes []string) string {
	sortedTypes := sortStrings(types)
	sortedTemplateHashes := sortStrings(templateHashes)
	return stableUUID("node", strings.Join(sortedTypes, "|"), name, strings.Join(sortedTemplateHashes, "|"))
}

func RelationshipUID(relationshipType, templateHash string, source, target domain.NodeSelector) string {
	return stableUUID(
		"relationship",
		relationshipType,
		templateHash,
		SelectorIdentity(source),
		SelectorIdentity(target),
	)
}

func SelectorIdentity(selector domain.NodeSelector) string {
	keys := make([]string, 0, len(selector.Attributes))
	for key := range selector.Attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := []string{"selector", selector.Type}
	for _, key := range keys {
		parts = append(parts, key, canonicalValue(selector.Attributes[key]))
	}

	return strings.Join(parts, "\x1f")
}

func stableUUID(parts ...string) string {
	hash := sha1.Sum([]byte(strings.Join(parts, "\x1f")))
	raw := hash[:16]

	raw[6] = (raw[6] & 0x0f) | 0x50
	raw[8] = (raw[8] & 0x3f) | 0x80

	encoded := hex.EncodeToString(raw)
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		encoded[0:8],
		encoded[8:12],
		encoded[12:16],
		encoded[16:20],
		encoded[20:32],
	)
}

func sortStrings(values []string) []string {
	output := append([]string(nil), values...)
	sort.Strings(output)
	return output
}

func canonicalValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprint(value)
		}
		return string(encoded)
	}
}
