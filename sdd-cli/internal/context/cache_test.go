package context

import "testing"

func TestFormatBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input int
		want  string
	}{
		{0, "0B"},
		{512, "512B"},
		{1023, "1023B"},
		{1024, "1KB"},
		{2048, "2KB"},
		{102400, "100KB"},
	}
	for _, tc := range tests {
		got := formatBytes(tc.input)
		if got != tc.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestMustParseInt64(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  int64
	}{
		{"0", 0},
		{"1234567890", 1234567890},
		{"-1", -1},
		{"", 0},        // invalid → 0
		{"abc", 0},     // invalid → 0
		{"1.5", 0},     // float → 0
	}
	for _, tc := range tests {
		got := mustParseInt64(tc.input)
		if got != tc.want {
			t.Errorf("mustParseInt64(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		bytes int
		want  int
	}{
		{0, 0},
		{4, 1},
		{400, 100},
		{4096, 1024},
	}
	for _, tc := range tests {
		got := estimateTokens(tc.bytes)
		if got != tc.want {
			t.Errorf("estimateTokens(%d) = %d, want %d", tc.bytes, got, tc.want)
		}
	}
}
