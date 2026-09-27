package tui

import (
	"reflect"
	"strings"
	"testing"
)

func stdFields() []Field {
	return []Field{{Label: "Alias:", Key: "alias"}, {Label: "Remote:", Key: "remote"}}
}

func TestFormInsertion(t *testing.T) {
	st := NewFormState("T", stdFields(), nil)
	st.HandleKey(KeyChar('a'))
	st.HandleKey(KeyChar('b'))
	if st.Values[0] != "ab" {
		t.Fatalf("Values[0] = %q, want %q", st.Values[0], "ab")
	}
	if st.Pos[0] != 2 {
		t.Fatalf("Pos[0] = %d, want 2", st.Pos[0])
	}
}

func TestFormBackspaceMiddle(t *testing.T) {
	st := NewFormState("T", stdFields(), nil)
	for _, r := range []rune{'a', 'b', 'c'} {
		st.HandleKey(KeyChar(r))
	}
	// caret at end (pos=3), move left twice -> pos=1
	st.HandleKey(KeyCode(Left))
	st.HandleKey(KeyCode(Left))
	st.HandleKey(KeyCode(Backspace))
	if st.Values[0] != "bc" {
		t.Fatalf("Values[0] = %q, want %q", st.Values[0], "bc")
	}
	if st.Pos[0] != 0 {
		t.Fatalf("Pos[0] = %d, want 0", st.Pos[0])
	}
}

func TestFormDeleteAtCaret(t *testing.T) {
	st := NewFormState("T", stdFields(), nil)
	for _, r := range []rune{'a', 'b', 'c'} {
		st.HandleKey(KeyChar(r))
	}
	// caret at end (pos=3), move left -> pos=2 (before 'c')
	st.HandleKey(KeyCode(Left))
	st.HandleKey(KeyCode(Delete))
	if st.Values[0] != "ab" {
		t.Fatalf("Values[0] = %q, want %q", st.Values[0], "ab")
	}
	if st.Pos[0] != 2 {
		t.Fatalf("Pos[0] = %d, want 2", st.Pos[0])
	}
}

func TestFormLeftRightClamp(t *testing.T) {
	st := NewFormState("T", stdFields(), nil)
	st.HandleKey(KeyChar('a'))  // pos=1
	st.HandleKey(KeyCode(Left)) // pos=0
	st.HandleKey(KeyCode(Left)) // clamp 0
	if st.Pos[0] != 0 {
		t.Fatalf("Pos[0] after double-left = %d, want 0", st.Pos[0])
	}
	st.HandleKey(KeyCode(Right)) // pos=1
	st.HandleKey(KeyCode(Right)) // clamp len=1
	if st.Pos[0] != 1 {
		t.Fatalf("Pos[0] after double-right = %d, want 1", st.Pos[0])
	}
}

func TestFormHomeEnd(t *testing.T) {
	st := NewFormState("T", stdFields(), nil)
	for _, r := range []rune{'a', 'b', 'c'} {
		st.HandleKey(KeyChar(r))
	}
	st.HandleKey(KeyCode(Home))
	if st.Pos[0] != 0 {
		t.Fatalf("Pos[0] after Home = %d, want 0", st.Pos[0])
	}
	st.HandleKey(KeyCode(End))
	if st.Pos[0] != 3 {
		t.Fatalf("Pos[0] after End = %d, want 3", st.Pos[0])
	}
}

func TestFormTabCyclic(t *testing.T) {
	st := NewFormState("T", stdFields(), nil)
	st.HandleKey(KeyCode(Tab))
	if st.Cursor != 1 {
		t.Fatalf("Cursor after Tab = %d, want 1", st.Cursor)
	}
	st.HandleKey(KeyCode(Tab))
	if st.Cursor != 0 {
		t.Fatalf("Cursor after 2nd Tab = %d, want 0", st.Cursor)
	}
	st.HandleKey(KeyCode(ShiftTab))
	if st.Cursor != 1 {
		t.Fatalf("Cursor after ShiftTab = %d, want 1", st.Cursor)
	}
	st.HandleKey(KeyCode(ShiftTab))
	if st.Cursor != 0 {
		t.Fatalf("Cursor after 2nd ShiftTab = %d, want 0", st.Cursor)
	}
}

func TestFormEnterBuildsResult(t *testing.T) {
	fields := []Field{
		{Label: "Alias:", Key: "alias"},
		{Label: "Remote:", Key: "remote"},
		{Label: "Key name:", Key: "keyname"},
	}
	st := NewFormState("T", fields, nil)
	st.HandleKey(KeyChar('x')) // alias = "x"
	st.HandleKey(KeyCode(Tab))
	// remote stays empty
	st.HandleKey(KeyCode(Tab))
	// keyname stays empty
	result, done := st.HandleKey(KeyCode(Enter))
	if !done {
		t.Fatalf("Enter not done")
	}
	want := map[string]string{"alias": "x", "remote": ""}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("Enter result = %#v, want %#v", result, want)
	}
}

func TestFormEscCancels(t *testing.T) {
	st := NewFormState("T", stdFields(), nil)
	result, done := st.HandleKey(KeyCode(Esc))
	if !done {
		t.Fatalf("Esc not done")
	}
	if result != nil {
		t.Fatalf("Esc result = %#v, want nil", result)
	}
}

func TestFormCyrillicCharIgnored(t *testing.T) {
	st := NewFormState("T", stdFields(), nil)
	before := st.Values[0]
	st.HandleKey(KeyChar('я'))
	if st.Values[0] != before {
		t.Fatalf("Values[0] changed to %q, want unchanged %q", st.Values[0], before)
	}
	if st.Pos[0] != 0 {
		t.Fatalf("Pos[0] = %d, want 0", st.Pos[0])
	}
}

func TestFormPrefill(t *testing.T) {
	initial := map[string]string{"alias": "foo", "remote": "bar"}
	st := NewFormState("T", stdFields(), initial)
	if st.Values[0] != "foo" || st.Values[1] != "bar" {
		t.Fatalf("Values = %#v, want [foo bar]", st.Values)
	}
	if st.Pos[0] != 3 || st.Pos[1] != 3 {
		t.Fatalf("Pos = %#v, want [3 3]", st.Pos)
	}
}

func TestFormViewScrollTruncates(t *testing.T) {
	long := "0123456789abcdefghijklmnopqrstuv" // 32 runes
	fields := []Field{{Label: "F", Key: "f"}}
	st := NewFormState("T", fields, map[string]string{"f": long})
	// Byte the caret is at end (pos=32). Render with small width to force scroll.
	lines := st.View(40, 10)

	if st.Scroll[0] != 3 {
		t.Fatalf("Scroll[0] = %d, want 3", st.Scroll[0])
	}
	// field_w = max(8, dw-8); dw = min(110, 38) = 38 -> field_w = 30
	wantSeg := "3456789abcdefghijklmnopqrstuv" // long[3:]
	if !strings.Contains(strings.Join(lines, "\n"), wantSeg) {
		t.Fatalf("View output missing visible segment %q", wantSeg)
	}
	if len([]rune(wantSeg)) > 30 {
		t.Fatalf("visible segment length %d exceeds field_w 30", len([]rune(wantSeg)))
	}
}
