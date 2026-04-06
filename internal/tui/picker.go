package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
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
