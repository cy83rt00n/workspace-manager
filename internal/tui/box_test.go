package tui

import (
	"reflect"
	"testing"
)

func TestDrawBoxGolden(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		title  string
		want   []string
	}{
		{"empty", 5, 3, "", []string{"┌───┐", "│   │", "└───┘"}},
		{"title", 10, 3, "Hi", []string{"┌── Hi ──┐", "│        │", "└────────┘"}},
		{"single-cell", 1, 1, "", []string{"┌"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DrawBox(tt.width, tt.height, tt.title)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("DrawBox(%d,%d,%q) = %#v, want %#v", tt.width, tt.height, tt.title, got, tt.want)
			}
		})
	}
}

func TestDrawBoxOutputWidth(t *testing.T) {
	for _, tt := range []struct {
		width, height int
	}{
		{8, 5},
		{20, 1},
		{1, 5},
		{3, 3},
	} {
		got := DrawBox(tt.width, tt.height, "t")
		if len(got) != tt.height {
			t.Fatalf("DrawBox height rows = %d, want %d", len(got), tt.height)
		}
		for i, line := range got {
			if n := len([]rune(line)); n != tt.width {
				t.Fatalf("DrawBox row %d len(runes)=%d, want %d (line=%q)", i, n, tt.width, line)
			}
		}
	}
}

func TestDrawBar(t *testing.T) {
	if got := DrawBar(5); got != "├───┤" {
		t.Fatalf("DrawBar(5) = %q, want %q", got, "├───┤")
	}
}

func TestDrawVDivider(t *testing.T) {
	if got := DrawVDivider(3); !reflect.DeepEqual(got, []string{"┬", "│", "┴"}) {
		t.Fatalf("DrawVDivider(3) = %#v, want %#v", got, []string{"┬", "│", "┴"})
	}
}

func TestFit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
	}{
		{"truncate", "abcdef", 4, "abcd"},
		{"negative", "ab", -1, ""},
		{"shorter", "ab", 10, "ab"},
		{"exact", "abcd", 4, "abcd"},
		{"empty", "", 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Fit(tt.input, tt.maxWidth); got != tt.want {
				t.Fatalf("Fit(%q,%d) = %q, want %q", tt.input, tt.maxWidth, got, tt.want)
			}
		})
	}
}
