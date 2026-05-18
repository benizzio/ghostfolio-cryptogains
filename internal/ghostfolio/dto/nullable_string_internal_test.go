package dto

import "testing"

// TestNullableStringUnmarshalJSONCoversBranches verifies nullable Ghostfolio
// string decoding across valid string, null, and invalid JSON inputs.
// Authored by: OpenCode
func TestNullableStringUnmarshalJSONCoversBranches(t *testing.T) {
	t.Parallel()

	var value NullableString
	if err := value.UnmarshalJSON([]byte(`"CHF"`)); err != nil {
		t.Fatalf("unmarshal string value: %v", err)
	}
	if value.String() != "CHF" {
		t.Fatalf("unexpected string value: %q", value.String())
	}

	if err := value.UnmarshalJSON([]byte("null")); err != nil {
		t.Fatalf("unmarshal null value: %v", err)
	}
	if value.String() != "" {
		t.Fatalf("expected null to clear nullable string, got %q", value.String())
	}

	if err := value.UnmarshalJSON([]byte("{")); err == nil {
		t.Fatalf("expected invalid JSON to fail nullable string decoding")
	}
}
