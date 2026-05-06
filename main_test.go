package main

import (
	"strings"
	"testing"
)

func TestHelpTextListsShortcuts(t *testing.T) {
	text := helpText()

	for _, want := range []string{
		"Usage:",
		"--help",
		"Ctrl+Shift+V",
		"Shift+PageUp",
		"Shift+PageDown",
		"Right click",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("helpText() missing %q in:\n%s", want, text)
		}
	}
}

func TestFallbackFontCandidatesIncludeCJKFallback(t *testing.T) {
	candidates := fallbackFontCandidates()

	found := false
	for _, path := range candidates {
		if strings.Contains(path, "DroidSansFallback") || strings.Contains(path, "NotoSansCJK") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("fallback candidates do not include a CJK fallback: %v", candidates)
	}
}

func TestLoadFontSetSkipsUnsupportedFallbackFonts(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("loadFontSet panicked on fallback font: %v", r)
		}
	}()

	if _, err := loadFontSet("", 16); err != nil {
		t.Fatalf("loadFontSet() error = %v", err)
	}
}
