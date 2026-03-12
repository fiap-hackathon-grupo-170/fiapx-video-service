package logger

import (
	"testing"
)

func TestNew_ValidLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			l, err := New(level)
			if err != nil {
				t.Fatalf("expected no error for level %q, got %v", level, err)
			}
			if l == nil {
				t.Fatal("expected non-nil logger")
			}
		})
	}
}

func TestNew_InvalidLevel(t *testing.T) {
	_, err := New("invalid-level")
	if err == nil {
		t.Fatal("expected error for invalid level")
	}
}
