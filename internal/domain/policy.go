package domain

import "strings"

type UpdatePolicy string

const (
	UpdatePolicyCreate        UpdatePolicy = "create"
	UpdatePolicyMerge         UpdatePolicy = "merge"
	UpdatePolicyMergeAtChange UpdatePolicy = "merge_at_change"
)

func NormalizeUpdatePolicy(raw string) UpdatePolicy {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(UpdatePolicyMerge):
		return UpdatePolicyMerge
	case string(UpdatePolicyMergeAtChange), "mergeatchange", "merge-at-change":
		return UpdatePolicyMergeAtChange
	default:
		return UpdatePolicyCreate
	}
}
