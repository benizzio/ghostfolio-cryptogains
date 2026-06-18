package text

import "testing"

// TestContainsAll verifies matching, missing-fragment, and empty-requirement
// behavior for the shared text predicate.
// Authored by: OpenCode
func TestContainsAll(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name  string
		value string
		parts []string
		want  bool
	}{
		{
			name:  "all present",
			value: "decimal policy scale exceeds maximum supported scale 64",
			parts: []string{"decimal policy", "scale", "64"},
			want:  true,
		},
		{
			name:  "missing part",
			value: "decimal policy scale exceeds maximum supported scale 64",
			parts: []string{"decimal policy", "missing"},
			want:  false,
		},
		{
			name:  "no parts",
			value: "decimal policy scale exceeds maximum supported scale 64",
			parts: nil,
			want:  true,
		},
	}

	for _, test := range tests {
		var test = test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var got = ContainsAll(test.value, test.parts...)
			if got != test.want {
				t.Fatalf("ContainsAll() = %v, want %v", got, test.want)
			}
		})
	}
}
