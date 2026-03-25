package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Table builds an aligned, pipe-delimited table. Add all rows before calling
// Print so that column widths are computed from actual data.
type Table struct {
	headers []string
	rows    [][]string
	widths  []int
}

func NewTable(headers ...string) *Table {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &Table{headers: headers, widths: widths}
}

// AddRow appends a data row and updates column widths.
func (t *Table) AddRow(cols ...string) {
	row := make([]string, len(t.headers))
	for i := range row {
		if i < len(cols) {
			row[i] = cols[i]
		}
		if len(row[i]) > t.widths[i] {
			t.widths[i] = len(row[i])
		}
	}
	t.rows = append(t.rows, row)
}

// Len returns the number of data rows.
func (t *Table) Len() int { return len(t.rows) }

// Print writes the table to stdout.
func (t *Table) Print() { t.PrintTo(os.Stdout) }

// PrintTo writes the table to w.
func (t *Table) PrintTo(w io.Writer) {
	printRow := func(cols []string) {
		fmt.Fprint(w, "|")
		for i, col := range cols {
			fmt.Fprintf(w, " %-*s |", t.widths[i], col)
		}
		fmt.Fprintln(w)
	}

	printRow(t.headers)

	seps := make([]string, len(t.headers))
	for i, width := range t.widths {
		seps[i] = strings.Repeat("-", width)
	}
	printRow(seps)

	for _, row := range t.rows {
		printRow(row)
	}
}
