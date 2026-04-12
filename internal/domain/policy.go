package domain

import "strings"

type UpdatePolicy string

const (
	UpdatePolicyCreate UpdatePolicy = "create"
	UpdatePolicyMerge  UpdatePolicy = "merge"
)

func NormalizeUpdatePolicy(raw string) UpdatePolicy {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(UpdatePolicyMerge):
		return UpdatePolicyMerge
	default:
		return UpdatePolicyCreate
	}
}
