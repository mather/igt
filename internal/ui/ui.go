package ui

import (
	"fmt"
	"io"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mather/igt/internal/template"
)

var (
	titleStyle = lipgloss.NewStyle().
			MarginLeft(2).
			Foreground(lipgloss.Color("62")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginLeft(2).
			MarginTop(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170"))
)

type item struct {
	template template.Template
	selected bool
}

func (i item) FilterValue() string {
	return i.template.GetDisplayName()
}

func (i item) Title() string {
	checkbox := "[ ]"
	if i.selected {
		checkbox = "[✓]"
	}
	return fmt.Sprintf("%s %s (%s)", checkbox, i.template.GetDisplayName(), i.template.Category)
}

func (i item) Description() string {
	return ""
}

type itemDelegate struct {
	list.DefaultDelegate
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := i.Title()

	var rendered string
	if index == m.Index() {
		rendered = selectedStyle.Render("> " + str)
	} else {
		rendered = lipgloss.NewStyle().PaddingLeft(2).Render(str)
	}

	fmt.Fprint(w, rendered)
}

func (d itemDelegate) Height() int {
	return 1
}

func (d itemDelegate) Spacing() int {
	return 0
}

type keyMap struct {
	Toggle key.Binding
	Confirm key.Binding
	Quit key.Binding
}

var keys = keyMap{
	Toggle: key.NewBinding(
		key.WithKeys(" ", "tab"),
		key.WithHelp("space/tab", "toggle selection"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Quit: key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "quit"),
	),
}

type Model struct {
	list     list.Model
	items    []item
	selected map[int]bool
	quitting bool
	cancelled bool
}

func NewModel(templates []template.Template) Model {
	// Group and sort templates by category (Language, Global, Community)
	sort.Slice(templates, func(i, j int) bool {
		orderI := template.CategoryOrder(templates[i].Category)
		orderJ := template.CategoryOrder(templates[j].Category)
		if orderI != orderJ {
			return orderI < orderJ
		}
		return templates[i].GetDisplayName() < templates[j].GetDisplayName()
	})

	items := make([]list.Item, len(templates))
	itemsData := make([]item, len(templates))
	for i, tmpl := range templates {
		itemsData[i] = item{template: tmpl, selected: false}
		items[i] = itemsData[i]
	}

	delegate := itemDelegate{list.NewDefaultDelegate()}

	l := list.New(items, delegate, 0, 0)
	l.Title = "Select gitignore templates"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle

	// Disable the default "enter to apply filter" help text
	l.KeyMap.AcceptWhileFiltering.SetEnabled(false)

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.Toggle,
			keys.Confirm,
		}
	}

	return Model{
		list:     l,
		items:    itemsData,
		selected: make(map[int]bool),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4)
		return m, nil

	case tea.KeyMsg:
		// When filtering is active
		if m.list.FilterState() == list.Filtering {
			// Esc key clears the filter (don't quit)
			if msg.String() == "esc" {
				m.list.ResetFilter()
				return m, nil
			}
			// Let the list handle the filter input
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		// Handle quit (only when not filtering)
		if key.Matches(msg, keys.Quit) {
			m.quitting = true
			m.cancelled = true
			return m, tea.Quit
		}

		// When filter is applied but not actively typing
		if m.list.FilterState() == list.FilterApplied {
			// Handle space or tab for toggle selection even with filter applied
			if msg.String() == " " || msg.String() == "tab" {
				var cmd tea.Cmd
				if idx := m.list.Index(); idx >= 0 && idx < len(m.list.VisibleItems()) {
					// Find the actual item in the full list
					visibleItem, ok := m.list.VisibleItems()[idx].(item)
					if ok {
						for i := range m.items {
							if m.items[i].template.Name == visibleItem.template.Name {
								m.items[i].selected = !m.items[i].selected
								m.selected[i] = m.items[i].selected

								// Update all list items
								items := make([]list.Item, len(m.items))
								for j := range m.items {
									items[j] = m.items[j]
								}
								cmd = m.list.SetItems(items)
								break
							}
						}
					}
				}
				return m, cmd
			}

			// Handle confirm
			if key.Matches(msg, keys.Confirm) {
				m.quitting = true
				return m, tea.Quit
			}

			// Let list handle other keys (arrow keys, etc.)
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		// Normal mode (no filter)
		switch {
		case key.Matches(msg, keys.Confirm):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.Toggle):
			if idx := m.list.Index(); idx >= 0 && idx < len(m.items) {
				m.items[idx].selected = !m.items[idx].selected
				m.selected[idx] = m.items[idx].selected

				// Update the list item
				items := m.list.Items()
				if idx < len(items) {
					items[idx] = m.items[idx]
					m.list.SetItems(items)
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var help string
	if m.list.FilterState() == list.Filtering || m.list.FilterState() == list.FilterApplied {
		help = helpStyle.Render(
			"space/tab: toggle • esc: clear filter • enter: confirm",
		)
	} else {
		help = helpStyle.Render(
			"space/tab: toggle • /: filter • enter: confirm • esc: quit",
		)
	}

	return m.list.View() + "\n" + help
}

func (m Model) SelectedTemplates() []template.Template {
	if m.cancelled {
		return nil
	}

	var selected []template.Template
	for idx := range m.selected {
		if idx < len(m.items) && m.items[idx].selected {
			selected = append(selected, m.items[idx].template)
		}
	}
	return selected
}

// Run starts the TUI and returns selected templates
func Run(templates []template.Template) ([]template.Template, error) {
	m := NewModel(templates)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}

	model, ok := finalModel.(Model)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	return model.SelectedTemplates(), nil
}
