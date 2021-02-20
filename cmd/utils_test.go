package cmd

import "testing"

func TestCheckParentSatisfies(t *testing.T) {
	tests := []struct {
		parentReference string
		childReference  string
		expected        bool
	}{
		{"1.0.0", "2.0.0", false},
		{"2.0.0", "1.0.0", false},
		{"1.2.3", "^1.0.0", true},
		{"2.0.0", "^1.2.3", false},
	}

	for _, tt := range tests {
		res := checkParentSatisfies(tt.parentReference, tt.childReference)
		if res != tt.expected {
			t.Fatalf("expected=%T, got=%T", tt.expected, res)
		}
	}
}
