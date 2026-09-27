package tui

import (
	"strings"
	"testing"
)

func TestHelpLinesContainsGolden(t *testing.T) {
	lines := HelpLines("/tmp/cfg", 60, 16)
	join := strings.Join(lines, "\n")

	for _, nav := range []string{
		"  Tab / ←→  Switch panels",
		"  ↑↓ / j k  Navigate",
		"  Enter     Execute",
		"  Esc       Back to left panel",
	} {
		if !containsLine(lines, nav) {
			t.Errorf("HelpLines missing nav line %q", nav)
		}
	}

	for _, want := range []string{
		"  F3    / m  Mount       F6    / u  Unmount   ",
		"  Configs:  /tmp/cfg",
		"  Min size: 60×16",
	} {
		if !containsLine(lines, want) {
			t.Errorf("HelpLines missing exact line %q", want)
		}
	}

	for _, sub := range []string{
		"F10   / q  Quit",
		"F1          Help",
	} {
		if !strings.Contains(join, sub) {
			t.Errorf("HelpLines output %q missing substring %q", join, sub)
		}
	}
}

func TestConfirmInput(t *testing.T) {
	tests := []struct {
		name         string
		choice       int
		key          Input
		wantChoice   int
		wantDone     bool
		wantAccepted bool
	}{
		{"enter-with-no", 1, KeyCode(Enter), 1, true, false},
		{"enter-with-yes", 0, KeyCode(Enter), 0, true, true},
		{"left-h", 1, KeyChar('h'), 0, false, false},
		{"right-l", 0, KeyChar('l'), 1, false, false},
		{"cyr-й-closes-no", 1, KeyChar('й'), 1, true, false},
		{"cyr-н-closes-yes", 1, KeyChar('н'), 0, true, true},
		{"lat-n-closes-no", 0, KeyChar('n'), 1, true, false},
		{"lat-y-closes-yes", 1, KeyChar('y'), 0, true, true},
		{"keycode-left", 1, KeyCode(Left), 0, false, false},
		{"keycode-right", 0, KeyCode(Right), 1, false, false},
		{"keycode-esc", 1, KeyCode(Esc), 1, true, false},
		{"keycode-tab-noop", 1, KeyCode(Tab), 1, false, false},
		{"unknown-char-noop", 0, KeyChar('x'), 0, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChoice, gotDone, gotAccepted := ConfirmInput(tt.choice, tt.key)
			if gotChoice != tt.wantChoice || gotDone != tt.wantDone || gotAccepted != tt.wantAccepted {
				t.Fatalf("ConfirmInput(%d,%v) = (%d,%v,%v), want (%d,%v,%v)",
					tt.choice, tt.key, gotChoice, gotDone, gotAccepted, tt.wantChoice, tt.wantDone, tt.wantAccepted)
			}
		})
	}
}

func TestTooSmallLines(t *testing.T) {
	lines := TooSmallLines(50, 10, 60, 16)
	join := strings.Join(lines, "\n")
	for _, want := range []string{
		"Terminal Too Small",
		"Terminal too small (50x10). Need 60x16+.",
		"Press q to quit, any key to retry.",
	} {
		if !strings.Contains(join, want) {
			t.Errorf("TooSmallLines output %q missing %q", join, want)
		}
	}
	if len(lines) != 5 {
		t.Fatalf("TooSmallLines len = %d, want 5", len(lines))
	}
}

func containsLine(lines []string, want string) bool {
	for _, l := range lines {
		if l == want {
			return true
		}
	}
	return false
}
