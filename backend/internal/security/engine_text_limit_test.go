package security

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitizeVulnerabilityTextFieldKeepsShortText(t *testing.T) {
	input := "short response"
	if got := sanitizeVulnerabilityTextField(input); got != input {
		t.Fatalf("expected short text to stay unchanged, got %q", got)
	}
}

func TestSanitizeVulnerabilityTextFieldTruncatesByBytesAndKeepsUTF8(t *testing.T) {
	input := strings.Repeat("中", 30000)
	got := sanitizeVulnerabilityTextField(input)

	if len(got) > 60*1024 {
		t.Fatalf("expected truncated text within byte limit, got %d bytes", len(got))
	}
	if !utf8.ValidString(got) {
		t.Fatal("expected truncated text to remain valid UTF-8")
	}
	if got == input {
		t.Fatal("expected long text to be truncated")
	}
}

