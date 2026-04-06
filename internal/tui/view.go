package tui

import (
	"fmt"
	"strings"
)

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
		start := maxInt(0, m.cursor-m.maxHeight/2)
		end := minInt(len(visible), start+m.maxHeight)
		start = maxInt(0, end-m.maxHeight)
		visible = visible[start:end]
	}

	for _, itemIdx := range visible {
		item := m.items[itemIdx]
		isCursor := m.filtered[minInt(m.cursor, len(m.filtered)-1)] == itemIdx

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
		sb.WriteString(line + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render(fmt.Sprintf("  %d/%d  ↑↓ navigate  enter select  esc quit", len(m.filtered), len(m.items))))

	return sb.String()
}

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
