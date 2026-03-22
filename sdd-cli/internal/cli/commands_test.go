package cli

import (
	"testing"
)

func TestValidateChangeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "add-auth", false},
		{"valid with numbers", "feat-123", false},
		{"empty", "", true},
		{"dot", ".", true},
		{"dotdot", "..", true},
		{"forward slash", "a/b", true},
		{"backslash", `a\b`, true},
		{"path traversal", "../etc/passwd", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateChangeName(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateChangeName(%q) error = %v, wantErr = %v", tc.input, err, tc.wantErr)
			}
		})
	}
}
