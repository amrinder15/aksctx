package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Item is a generic selectable item in the picker.
type Item struct {
	Label       string // displayed in the list
	Description string // dimmed secondary text
	Value       any    // arbitrary payload returned on selection
}

// Result is returned after the user picks an item or cancels.
type Result struct {
	Selected *Item
	Aborted  bool
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00B4D8")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00B4D8")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666677"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00B4D8"))

	matchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD166")).
			Bold(true)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00B4D8")).
			Padding(0, 1).
			MarginBottom(1)
)

type model struct {
	title     string
	items     []Item
	filtered  []int // indices into items
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
	// Clamp cursor after filter change
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
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
		if strings.Contains(strings.ToLower(item.Label), query) ||
			strings.Contains(strings.ToLower(item.Description), query) {
			result = append(result, i)
		}
	}
	m.filtered = result
}

func (m model) View() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render(m.title))
	sb.WriteString("\n")
	sb.WriteString(inputStyle.Render(m.input.View()))
	sb.WriteString("\n")

	if len(m.filtered) == 0 {
		sb.WriteString(dimStyle.Render("  no matches"))
		sb.WriteString("\n")
		return sb.String()
	}

	query := strings.ToLower(m.input.Value())
	visible := m.filtered
	if len(visible) > m.maxHeight {
		// Scroll window around cursor
		start := max(0, m.cursor-m.maxHeight/2)
		end := min(len(visible), start+m.maxHeight)
		start = max(0, end-m.maxHeight)
		visible = visible[start:end]
		// Adjust cursor for the visible window
	}

	for relIdx, itemIdx := range visible {
		item := m.items[itemIdx]
		isCursor := m.filtered[min(m.cursor, len(m.filtered)-1)] == itemIdx

		cursor := "  "
		if isCursor {
			cursor = cursorStyle.Render("▶ ")
		}

		label := highlightMatch(item.Label, query)
		desc := ""
		if item.Description != "" {
			desc = "  " + dimStyle.Render(item.Description)
		}

		line := cursor + label + desc
		if isCursor {
			line = cursor + selectedStyle.Render(item.Label) + desc
		}
		_ = relIdx
		sb.WriteString(line + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render(fmt.Sprintf("  %d/%d  ↑↓ navigate  enter select  esc quit", len(m.filtered), len(m.items))))

	return sb.String()
}

// highlightMatch bolds matched substrings in the label.
func highlightMatch(label, query string) string {
	if query == "" {
		return label
	}
	lower := strings.ToLower(label)
	idx := strings.Index(lower, query)
	if idx < 0 {
		return label
	}
	return label[:idx] + matchStyle.Render(label[idx:idx+len(query)]) + label[idx+len(query):]
}

// Run launches the TUI picker and returns the Result.
func Run(title string, items []Item) (Result, error) {
	if len(items) == 0 {
		return Result{Aborted: true}, fmt.Errorf("no items to display")
	}

	m := newModel(title, items)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return Result{Aborted: true}, err
	}

	fm := final.(model)
	if fm.result == nil {
		return Result{Aborted: true}, nil
	}
	return *fm.result, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
