package typenames

import "testing"

func TestAllReturnsCanonicalTypeNames(t *testing.T) {
	got := All()
	want := []string{
		"number",
		"string",
		"boolean",
		"nil",
		"list",
		"record",
		"error",
		"function",
		"native-function",
		"code",
		"mutation",
	}

	if len(got) != len(want) {
		t.Fatalf("len(All()) = %d, want %d", len(got), len(want))
	}

	seen := make(map[string]struct{}, len(got))

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("All()[%d] = %q, want %q", i, got[i], want[i])
		}

		if _, exists := seen[got[i]]; exists {
			t.Fatalf("duplicate type name %q", got[i])
		}

		seen[got[i]] = struct{}{}
	}
}
