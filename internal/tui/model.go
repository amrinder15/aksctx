package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	title     string
	items     []Item
	filtered  []int
	input     textinput.Model
	cursor    int
	result    *Result
	maxHeight int
}

func newModel(title string, items []Item) model {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	m := model{
		title:     title,
		items:     items,
		input:     ti,
		maxHeight: 12,
	}
	m.filtered = m.allIndices()
	return m
}

func (m model) allIndices() []int {
	idx := make([]int, len(m.items))
	for i := range idx {
		idx[i] = i
	}
	return idx
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.result = &Result{Aborted: true}
			return m, tea.Quit

		case "enter":
			if len(m.filtered) > 0 {
				selected := m.items[m.filtered[m.cursor]]
				m.result = &Result{Selected: &selected}
			}
			return m, tea.Quit

		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.applyFilter()
	if m.cursor >= len(m.filtered) {
		m.cursor = maxInt(0, len(m.filtered)-1)
	}
	return m, cmd
}

func (m *model) applyFilter() {
	query := strings.ToLower(m.input.Value())
	if query == "" {
		m.filtered = m.allIndices()
		return
	}

	var result []int
	for i, item := range m.items {
		if strings.Contains(strings.ToLower(item.Label), query) || strings.Contains(strings.ToLower(item.Description), query) {
			result = append(result, i)
		}
	}
	m.filtered = result
}
