package datatable

import (
	"github.com/charmbracelet/lipgloss"
)

// Align of a column's content within its width.
type Align int

// Alignments.
const (
	AlignLeft Align = iota
	AlignRight
)

// Column of a table.
//
// A column is either fixed or flexible. A fixed column gets exactly Width. A
// flexible one gets a share of whatever is left over, weighted by Flex, and
// never less than MinWidth.
type Column struct {
	// Title shown in the header.
	Title string
	// Width of a fixed column.
	Width int
	// Flex is a flexible column's share of the remaining width. Zero means the
	// column is fixed.
	Flex int
	// MinWidth a flexible column will not shrink below.
	MinWidth int
	// Align of the content.
	Align Align
	// Priority orders which columns are given up when even the minimums will
	// not fit. Higher goes first; zero is never dropped.
	//
	// Dropping a column outright beats squeezing six of them into four
	// characters each, which is what makes a narrow terminal readable rather
	// than merely non-overlapping.
	Priority int
	// Style the column's cells default to. Cells may override it.
	Style *lipgloss.Style
}

// fixed reports whether the column takes a fixed width.
func (c Column) fixed() bool {
	return c.Flex <= 0
}

// floor of a column, below which it is not worth rendering.
func (c Column) floor() int {
	if c.fixed() {
		return c.Width
	}

	if c.MinWidth > 0 {
		return c.MinWidth
	}

	return 1
}
