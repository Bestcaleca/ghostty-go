package terminal

import "testing"

func TestGetSelectedTextClampsOutOfBoundsColumns(t *testing.T) {
	term := New(2, 5)
	for _, ch := range "hello" {
		term.Print(ch)
	}

	term.SelectionStart(0, -3, SelectionChar)
	term.SelectionUpdate(0, 99)

	if got := term.GetSelectedText(); got != "hello" {
		t.Fatalf("GetSelectedText() = %q, want %q", got, "hello")
	}
}
