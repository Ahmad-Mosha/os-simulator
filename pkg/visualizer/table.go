package visualizer

import (
	"strings"
)

// Table renders formatted ASCII tables
type Table struct {
	Headers []string
	Rows    [][]string
	Align   []string // "left", "right", "center"
}

// NewTable initializes a new table
func NewTable(headers ...string) *Table {
	return &Table{
		Headers: headers,
		Rows:    make([][]string, 0),
		Align:   make([]string, len(headers)),
	}
}

// AddRow adds a row of values
func (t *Table) AddRow(cols ...string) {
	t.Rows = append(t.Rows, cols)
}

// SetAlignment sets column alignment
func (t *Table) SetAlignment(align ...string) {
	t.Align = align
}

// Render returns the formatted table string
func (t *Table) Render() string {
	numCols := len(t.Headers)
	colWidths := make([]int, numCols)

	for i, h := range t.Headers {
		colWidths[i] = len(StripANSI(h))
	}

	for _, row := range t.Rows {
		for i, cell := range row {
			if i < numCols {
				l := len(StripANSI(cell))
				if l > colWidths[i] {
					colWidths[i] = l
				}
			}
		}
	}

	var sb strings.Builder

	// Top border
	sb.WriteString(FgCyan + "┌")
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w+2))
		if i < numCols-1 {
			sb.WriteString("┬")
		}
	}
	sb.WriteString("┐\n" + Reset)

	// Headers
	sb.WriteString(FgCyan + "│" + Reset)
	for i, h := range t.Headers {
		sb.WriteString(" " + Bold + FgHiWhite + padString(h, colWidths[i], "center") + Reset + " " + FgCyan + "│" + Reset)
	}
	sb.WriteString("\n")

	// Header separator
	sb.WriteString(FgCyan + "├")
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w+2))
		if i < numCols-1 {
			sb.WriteString("┼")
		}
	}
	sb.WriteString("┤\n" + Reset)

	// Rows
	for _, row := range t.Rows {
		sb.WriteString(FgCyan + "│" + Reset)
		for i := 0; i < numCols; i++ {
			val := ""
			if i < len(row) {
				val = row[i]
			}
			align := "left"
			if i < len(t.Align) && t.Align[i] != "" {
				align = t.Align[i]
			}
			sb.WriteString(" " + padString(val, colWidths[i], align) + " " + FgCyan + "│" + Reset)
		}
		sb.WriteString("\n")
	}

	// Bottom border
	sb.WriteString(FgCyan + "└")
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w+2))
		if i < numCols-1 {
			sb.WriteString("┴")
		}
	}
	sb.WriteString("┘\n" + Reset)

	return sb.String()
}

func padString(s string, width int, align string) string {
	rawLen := len(StripANSI(s))
	if rawLen >= width {
		return s
	}
	pad := width - rawLen

	switch align {
	case "right":
		return strings.Repeat(" ", pad) + s
	case "center":
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	default: // left
		return s + strings.Repeat(" ", pad)
	}
}
