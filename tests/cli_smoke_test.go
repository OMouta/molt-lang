package integration_test

import "testing"

func TestCLISmokeExamples(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantOutput string
	}{
		{
			name:       "basic mutation",
			path:       "examples/basic/basic_mutation.molt",
			wantOutput: "5\n6\n",
		},
		{
			name:       "compare worlds",
			path:       "examples/basic/compare_worlds.molt",
			wantOutput: "5\n6\n",
		},
		{
			name:       "import export",
			path:       "examples/import_export/main.molt",
			wantOutput: "40\n42\n",
		},
		{
			name:       "records",
			path:       "examples/basic/records.molt",
			wantOutput: "record { name: \"molt\", stats: record { runs: 4 }, age: 2 }\nmolt\n4\n2\n[\"name\", \"stats\", \"age\"]\n[\"molt\", record { runs: 4 }, 2]\ntrue\n3\nrecord\n",
		},
		{
			name:       "match",
			path:       "examples/basic/match.molt",
			wantOutput: "two\nmolt\nno\n",
		},
		{
			name:       "error values",
			path:       "examples/errors/error_values.molt",
			wantOutput: "error\nmissing file\nnote.txt\n[\"message\", \"data\"]\nerror {\n  message: \"missing file\",\n  data: record { path: \"note.txt\" }\n}\n",
		},
		{
			name:       "try catch",
			path:       "examples/errors/try_catch.molt",
			wantOutput: "[\"helper failed\", \"import\"]\nlen expects list, string, record, or error, got \"number\"\nname\n",
		},
		{
			name:       "variant gallery",
			path:       "examples/basic/variant_gallery.molt",
			wantOutput: "[6, 7, \"code\"]\n",
		},
		{
			name:       "while loop",
			path:       "examples/loops/while_loop.molt",
			wantOutput: "3\n",
		},
		{
			name:       "for loop",
			path:       "examples/loops/for_loop.molt",
			wantOutput: "6\n[\"o\", \"k\"]\n",
		},
		{
			name:       "break continue loop",
			path:       "examples/loops/break_continue.molt",
			wantOutput: "[1, 3]\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr := runCLIExample(t, tc.path)
			if stdout != tc.wantOutput {
				t.Fatalf("stdout = %q, want %q", stdout, tc.wantOutput)
			}

			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
		})
	}
}
