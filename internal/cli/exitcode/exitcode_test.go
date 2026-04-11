package exitcode

import "testing"

func TestValuesRemainStable(t *testing.T) {
	expected := map[string]int{
		"Success":     0,
		"Usage":       1,
		"SourceIO":    2,
		"Diagnostics": 3,
		"Runtime":     4,
		"NeedsFormat": 5,
		"Internal":    10,
	}

	actual := map[string]int{
		"Success":     Success,
		"Usage":       Usage,
		"SourceIO":    SourceIO,
		"Diagnostics": Diagnostics,
		"Runtime":     Runtime,
		"NeedsFormat": NeedsFormat,
		"Internal":    Internal,
	}

	if len(actual) != len(expected) {
		t.Fatalf("unexpected exit code count: got %d want %d", len(actual), len(expected))
	}

	seen := make(map[int]string, len(actual))

	for name, want := range expected {
		got, ok := actual[name]
		if !ok {
			t.Fatalf("missing exit code %q", name)
		}

		if got != want {
			t.Fatalf("%s = %d, want %d", name, got, want)
		}

		if other, exists := seen[got]; exists {
			t.Fatalf("exit code %d is used by both %s and %s", got, other, name)
		}

		seen[got] = name
	}
}
