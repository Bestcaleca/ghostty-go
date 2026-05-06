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
