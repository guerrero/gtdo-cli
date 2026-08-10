package cli

import (
	"reflect"
	"testing"
)

func TestAddCompleterUsesCurrentSigilWord(t *testing.T) {
	c := addCompleter{Candidates: addCandidates{
		Contexts: []string{"@home", "@office"},
		Projects: []string{"+gtdo"},
	}}

	got, consumed := c.Do([]rune("Call @o"), len([]rune("Call @o")))
	if consumed != 2 {
		t.Fatalf("consumed = %d, want 2", consumed)
	}
	want := [][]rune{[]rune("ffice")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates = %q, want %q", got, want)
	}
}

func TestAddCompleterUsesProjectPrefixAndSortsSuffixes(t *testing.T) {
	c := addCompleter{Candidates: addCandidates{
		Contexts: []string{"@home"},
		Projects: []string{"+gtdo", "+go", "+code"},
	}}

	got, consumed := c.Do([]rune("Call +g"), len([]rune("Call +g")))
	if consumed != 2 {
		t.Fatalf("consumed = %d, want 2", consumed)
	}
	want := [][]rune{[]rune("o"), []rune("tdo")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates = %q, want %q", got, want)
	}
}

func TestAddCompleterIgnoresEmptyAndUnknownWords(t *testing.T) {
	c := addCompleter{Candidates: addCandidates{
		Contexts: []string{"@home"},
		Projects: []string{"+gtdo"},
	}}

	for _, tc := range []struct {
		name string
		line string
	}{
		{name: "plain word", line: "Call"},
		{name: "empty word", line: "Call "},
		{name: "unknown context", line: "Call @missing"},
		{name: "unknown project", line: "Call +missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, consumed := c.Do([]rune(tc.line), len([]rune(tc.line)))
			if consumed != 0 {
				t.Fatalf("consumed = %d, want 0", consumed)
			}
			if got != nil {
				t.Fatalf("candidates = %q, want nil", got)
			}
		})
	}
}

func TestAddCompleterUsesCursorWithinToken(t *testing.T) {
	c := addCompleter{Candidates: addCandidates{
		Contexts: []string{"@office"},
	}}

	line := []rune("Call @of now")
	got, consumed := c.Do(line, len([]rune("Call @of")))
	if consumed != 3 {
		t.Fatalf("consumed = %d, want 3", consumed)
	}
	want := [][]rune{[]rune("fice")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates = %q, want %q", got, want)
	}
}

func TestSelectorModelFiltersAndToggles(t *testing.T) {
	m := newSelectorModel([]string{"@home", "@office", "@phone"})
	m.SetQuery("of")
	if got := m.Visible(); !reflect.DeepEqual(got, []string{"@office"}) {
		t.Fatalf("visible = %v, want [@office]", got)
	}
	m.Toggle()
	if got := m.Values(); !reflect.DeepEqual(got, []string{"@office"}) {
		t.Fatalf("values = %v, want [@office]", got)
	}

	m.SetQuery("")
	if got := m.Visible(); !reflect.DeepEqual(got, []string{"@home", "@office", "@phone"}) {
		t.Fatalf("visible after clearing = %v, want all options", got)
	}
}

func TestSelectorModelMovesAndClampsAfterFiltering(t *testing.T) {
	m := newSelectorModel([]string{"@home", "@office", "@phone"})
	m.Move(2)
	if m.Cursor != 2 {
		t.Fatalf("cursor after move = %d, want 2", m.Cursor)
	}
	m.Move(10)
	if m.Cursor != 2 {
		t.Fatalf("cursor after positive clamp = %d, want 2", m.Cursor)
	}
	m.Move(-10)
	if m.Cursor != 0 {
		t.Fatalf("cursor after negative clamp = %d, want 0", m.Cursor)
	}

	m.SetQuery("phone")
	if m.Cursor != 0 {
		t.Fatalf("cursor after filtering = %d, want 0", m.Cursor)
	}
	m.Move(-1)
	if m.Cursor != 0 {
		t.Fatalf("cursor after empty backward move = %d, want 0", m.Cursor)
	}

	m.SetQuery("missing")
	if m.Cursor != 0 {
		t.Fatalf("cursor after no-match filter = %d, want 0", m.Cursor)
	}
	if got := m.Visible(); got != nil {
		t.Fatalf("visible for no-match query = %v, want nil", got)
	}
	m.Toggle()
	if got := m.Values(); len(got) != 0 {
		t.Fatalf("values after toggling no match = %v, want empty", got)
	}
}

func TestSelectorModelReturnsSelectedValuesInOptionOrder(t *testing.T) {
	m := newSelectorModel([]string{"@home", "@office", "@phone"})
	m.Move(2)
	m.Toggle()
	m.Move(-2)
	m.Toggle()

	want := []string{"@home", "@phone"}
	if got := m.Values(); !reflect.DeepEqual(got, want) {
		t.Fatalf("values = %v, want %v", got, want)
	}
}

func TestSelectorModelQueryIsCaseSensitiveSubstring(t *testing.T) {
	m := newSelectorModel([]string{"@home", "@Office"})
	m.SetQuery("of")
	if got := m.Visible(); got != nil {
		t.Fatalf("visible for lowercase query = %v, want nil", got)
	}
	m.SetQuery("Off")
	if got := m.Visible(); !reflect.DeepEqual(got, []string{"@Office"}) {
		t.Fatalf("visible for matching query = %v, want [@Office]", got)
	}
}
