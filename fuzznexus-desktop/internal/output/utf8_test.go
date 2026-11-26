package output

import (
	"testing"
	"unicode/utf8"
)

func TestSanitizeUTF8TruncatesAtRuneBoundary(t *testing.T) {
	input := "abcژ"
	got := sanitizeUTF8(input, 4)
	if got != input {
		t.Fatalf("sanitizeUTF8() = %q, want %q", got, input)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("sanitizeUTF8() returned invalid UTF-8: %q", got)
	}

	if got := sanitizeUTF8(input, 3); got != "abc" {
		t.Fatalf("sanitizeUTF8() = %q, want %q", got, "abc")
	}
	if got := sanitizeUTF8(input, 0); got != "" {
		t.Fatalf("sanitizeUTF8() with zero limit = %q, want empty", got)
	}
}
