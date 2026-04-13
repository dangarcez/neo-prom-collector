package domain

import "testing"

func TestNormalizeUpdatePolicy(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected UpdatePolicy
	}{
		{
			name:     "defaults to create",
			input:    "",
			expected: UpdatePolicyCreate,
		},
		{
			name:     "merge remains merge",
			input:    "merge",
			expected: UpdatePolicyMerge,
		},
		{
			name:     "snake case merge at change",
			input:    "merge_at_change",
			expected: UpdatePolicyMergeAtChange,
		},
		{
			name:     "camel case merge at change",
			input:    "mergeAtChange",
			expected: UpdatePolicyMergeAtChange,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := NormalizeUpdatePolicy(testCase.input)
			if actual != testCase.expected {
				t.Fatalf("expected %q, got %q", testCase.expected, actual)
			}
		})
	}
}
