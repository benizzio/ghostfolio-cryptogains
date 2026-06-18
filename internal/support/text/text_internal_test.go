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

// TestContainsASCIILetter verifies positive, negative, and non-ASCII-only
// inputs for the shared ASCII letter predicate.
// Authored by: OpenCode
func TestContainsASCIILetter(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "uppercase ASCII letter",
			value: "123A",
			want:  true,
		},
		{
			name:  "lowercase ASCII letter",
			value: "123a",
			want:  true,
		},
		{
			name:  "no ASCII letter",
			value: "123-_=",
			want:  false,
		},
		{
			name:  "non ASCII letter only",
			value: "\u00e9",
			want:  false,
		},
	}

	for _, test := range tests {
		var test = test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var got = ContainsASCIILetter(test.value)
			if got != test.want {
				t.Fatalf("ContainsASCIILetter() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestContainsASCIIDigit verifies positive, negative, and non-ASCII-only
// inputs for the shared ASCII digit predicate.
// Authored by: OpenCode
func TestContainsASCIIDigit(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "ASCII digit",
			value: "abc1",
			want:  true,
		},
		{
			name:  "no ASCII digit",
			value: "abc-_=",
			want:  false,
		},
		{
			name:  "non ASCII digit only",
			value: "\u0663",
			want:  false,
		},
	}

	for _, test := range tests {
		var test = test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var got = ContainsASCIIDigit(test.value)
			if got != test.want {
				t.Fatalf("ContainsASCIIDigit() = %v, want %v", got, test.want)
			}
		})
	}
}
