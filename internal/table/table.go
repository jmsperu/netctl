package table

import (
	"fmt"
	"strings"
)

type Table struct {
	Headers []string
	Rows    [][]string
}

func New(headers ...string) *Table {
	return &Table{Headers: headers}
}

func (t *Table) AddRow(cols ...string) {
	t.Rows = append(t.Rows, cols)
}

func (t *Table) Render() {
	if len(t.Headers) == 0 {
		return
	}
	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = len(h)
	}
	for _, row := range t.Rows {
		for i, col := range row {
			if i < len(widths) && len(col) > widths[i] {
				widths[i] = len(col)
			}
		}
	}
	printRow(t.Headers, widths)
	sep := make([]string, len(widths))
	for i, w := range widths {
		sep[i] = strings.Repeat("─", w)
	}
	printRow(sep, widths)
	for _, row := range t.Rows {
		for len(row) < len(widths) {
			row = append(row, "")
		}
		printRow(row, widths)
	}
}

func printRow(cols []string, widths []int) {
	parts := make([]string, len(cols))
	for i, col := range cols {
		w := 0
		if i < len(widths) {
			w = widths[i]
		}
		parts[i] = fmt.Sprintf("%-*s", w, col)
	}
	fmt.Println(strings.Join(parts, "  "))
}
