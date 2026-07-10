package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mather/igt/internal/template"
)

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "/":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// TestFilterThenToggleSelectsItem is a regression test for a bug where
// pressing Space to toggle a selection right after applying a filter
// (typing a query and pressing Enter) discarded the list's filter
// re-computation command, leaving the visible list empty ("No items").
func TestFilterThenToggleSelectsItem(t *testing.T) {
	templates := []template.Template{
		{Name: "Go", Category: template.CategoryLanguage},
		{Name: "Node", Category: template.CategoryLanguage},
		{Name: "Godot", Category: template.CategoryLanguage},
	}

	m := NewModel(templates)
	m.list.SetWidth(80)
	m.list.SetHeight(20)

	// Enter filter mode and type a query that matches "Go" and "Godot".
	updated, _ := m.Update(keyMsg("/"))
	m = updated.(Model)
	for _, r := range "Go" {
		updated, _ = m.Update(keyMsg(string(r)))
		m = updated.(Model)
	}

	// Apply the filter.
	updated, cmd := m.Update(keyMsg("enter"))
	m = updated.(Model)
	if cmd != nil {
		if msg := cmd(); msg != nil {
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
	}

	if got := m.list.FilterState(); got != list.FilterApplied {
		t.Fatalf("expected filter to be applied, got %v", got)
	}
	if len(m.list.VisibleItems()) == 0 {
		t.Fatalf("expected filtered items to be visible before toggling")
	}

	// Toggle selection on the first filtered item and drain the resulting cmd.
	updated, cmd = m.Update(keyMsg(" "))
	m = updated.(Model)
	if cmd != nil {
		if msg := cmd(); msg != nil {
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
	}

	if len(m.list.VisibleItems()) == 0 {
		t.Fatalf("filtered items disappeared after toggling selection")
	}
	if strings.Contains(m.View(), "No items") {
		t.Fatalf("view shows \"No items\" after toggling selection:\n%s", m.View())
	}

	selected := m.SelectedTemplates()
	if len(selected) != 1 {
		t.Fatalf("expected exactly 1 selected template, got %d: %v", len(selected), selected)
	}
}
