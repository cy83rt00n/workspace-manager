package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// testANSIProfile — профиль терменв ANSI (значение 2) для тестов без TTY:
// заставляет lipgloss выдавать ANSI-escape даже в неинтерактивном окружении.
const testANSIProfile = 2

func TestColorConstants(t *testing.T) {
	if MinW != 60 {
		t.Errorf("MinW = %d, want 60", MinW)
	}
	if MinH != 16 {
		t.Errorf("MinH = %d, want 16", MinH)
	}
	if got := len([]rune(SpinnerChars)); got != 10 {
		t.Errorf("len([]rune(SpinnerChars)) = %d, want 10", got)
	}
}

func TestPairProducesANSIEscape(t *testing.T) {
	lipgloss.SetColorProfile(testANSIProfile)
	for i := 1; i <= 10; i++ {
		got := Pair(i).Render("x")
		if !strings.Contains(got, "\x1b[") {
			t.Errorf("Pair(%d).Render = %q, want an ANSI escape", i, got)
		}
	}
}

func TestPairOutOfRangeIsZeroStyle(t *testing.T) {
	lipgloss.SetColorProfile(testANSIProfile)
	for _, i := range []int{0, 11, -1} {
		got := Pair(i).Render("x")
		if strings.Contains(got, "\x1b[") {
			t.Errorf("Pair(%d).Render = %q, want no styling (zero style)", i, got)
		}
	}
}
