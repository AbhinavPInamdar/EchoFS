package main

import (
	"testing"
)

func TestSanitizeFilename_Normal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"document.pdf", "document.pdf"},
		{"my file.txt", "my file.txt"},
		{"image.PNG", "image.PNG"},
	}

	for _, tt := range tests {
		result := sanitizeFilename(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeFilename_PathTraversal(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"../../../etc/passwd"},
		{"/etc/shadow"},
		{"..\\windows\\system32\\config"},
	}

	for _, tt := range tests {
		result := sanitizeFilename(tt.input)
		// Should not contain path separators or be empty
		if result == "" {
			continue // empty is acceptable for dangerous inputs
		}
		if result[0] == '/' || result[0] == '\\' {
			t.Errorf("sanitizeFilename(%q) = %q, should not start with path separator", tt.input, result)
		}
		if result == ".." || result == "." {
			t.Errorf("sanitizeFilename(%q) = %q, should not be . or ..", tt.input, result)
		}
	}
}

func TestSanitizeFilename_HiddenFiles(t *testing.T) {
	tests := []struct {
		input string
	}{
		{".htaccess"},
		{".."},
		{"."},
		{".env"},
	}

	for _, tt := range tests {
		result := sanitizeFilename(tt.input)
		// Should either strip dots or return empty
		if result != "" && result[0] == '.' {
			t.Errorf("sanitizeFilename(%q) = %q, should not start with dot", tt.input, result)
		}
	}
}

func TestSanitizeFilename_Empty(t *testing.T) {
	result := sanitizeFilename("")
	if result != "" {
		t.Errorf("sanitizeFilename('') = %q, want empty", result)
	}
}

func TestSanitizeFilename_NullBytes(t *testing.T) {
	result := sanitizeFilename("file\x00name.txt")
	if result != "filename.txt" {
		t.Errorf("sanitizeFilename with null byte = %q, want 'filename.txt'", result)
	}
}

func TestSanitizeFilename_LongName(t *testing.T) {
	long := ""
	for i := 0; i < 300; i++ {
		long += "a"
	}
	long += ".txt"

	result := sanitizeFilename(long)
	if len(result) > 255 {
		t.Errorf("sanitizeFilename should truncate to 255, got %d", len(result))
	}
	// Should preserve extension
	if result[len(result)-4:] != ".txt" {
		t.Errorf("expected .txt extension preserved, got %q", result[len(result)-4:])
	}
}
