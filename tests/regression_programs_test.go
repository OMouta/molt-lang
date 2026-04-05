package integration_test

import "testing"

func TestRegressionPrograms(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantValue  string
		wantOutput string
	}{
		{
			name:       "basic mutation example",
			path:       "examples/basic_mutation.molt",
			wantValue:  "nil",
			wantOutput: "5\n6\n",
		},
		{
			name:       "compare worlds example",
			path:       "examples/compare_worlds.molt",
			wantValue:  "nil",
			wantOutput: "5\n6\n",
		},
		{
			name:       "variant gallery example",
			path:       "examples/variant_gallery.molt",
			wantValue:  "nil",
			wantOutput: "[6, 7, \"code\"]\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, output := mustExecuteFile(t, tc.path)
			expectShownValue(t, value, tc.wantValue)
			if output != tc.wantOutput {
				t.Fatalf("output = %q, want %q", output, tc.wantOutput)
			}
		})
	}
}
