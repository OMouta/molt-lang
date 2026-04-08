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
			path:       "examples/basic/basic_mutation.molt",
			wantValue:  "nil",
			wantOutput: "5\n6\n",
		},
		{
			name:       "compare worlds example",
			path:       "examples/basic/compare_worlds.molt",
			wantValue:  "nil",
			wantOutput: "5\n6\n",
		},
		{
			name:       "import export example",
			path:       "examples/import_export/main.molt",
			wantValue:  "nil",
			wantOutput: "40\n42\n",
		},
		{
			name:       "records example",
			path:       "examples/basic/records.molt",
			wantValue:  "nil",
			wantOutput: "record { name: \"molt\", stats: record { runs: 3 } }\nmolt\n3\n[\"name\", \"stats\"]\n[\"molt\", record { runs: 3 }]\ntrue\n2\nrecord\nAge not found in profile.\n",
		},
		{
			name:       "variant gallery example",
			path:       "examples/basic/variant_gallery.molt",
			wantValue:  "nil",
			wantOutput: "[6, 7, \"code\"]\n",
		},
		{
			name:       "while loop example",
			path:       "examples/loops/while_loop.molt",
			wantValue:  "nil",
			wantOutput: "3\n",
		},
		{
			name:       "for loop example",
			path:       "examples/loops/for_loop.molt",
			wantValue:  "nil",
			wantOutput: "6\n[\"o\", \"k\"]\n",
		},
		{
			name:       "break continue example",
			path:       "examples/loops/break_continue.molt",
			wantValue:  "nil",
			wantOutput: "[1, 3]\n",
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
