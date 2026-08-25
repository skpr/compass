package datatable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWidths_FixedColumnsGetExactlyTheirWidth(t *testing.T) {
	widths := resolveWidths([]Column{
		{Title: "a", Width: 10},
		{Title: "b", Width: 5},
	}, 100)

	assert.Equal(t, []int{10, 5}, widths)
}

func TestResolveWidths_FlexTakesWhatIsLeft(t *testing.T) {
	widths := resolveWidths([]Column{
		{Title: "flex", Flex: 1, MinWidth: 5},
		{Title: "fixed", Width: 10},
	}, 40)

	// 40 total, less the 10 fixed and the 2 gap.
	assert.Equal(t, []int{28, 10}, widths)
	assert.Equal(t, 40, sum(widths)+gap)
}

func TestResolveWidths_FlexSharesByWeight(t *testing.T) {
	widths := resolveWidths([]Column{
		{Title: "wide", Flex: 3, MinWidth: 1},
		{Title: "narrow", Flex: 1, MinWidth: 1},
	}, 42)

	require.Len(t, widths, 2)
	assert.Equal(t, 40, widths[0]+widths[1])
	assert.Greater(t, widths[0], widths[1]*2)
}

func TestResolveWidths_NeverBelowMinWidth(t *testing.T) {
	widths := resolveWidths([]Column{
		{Title: "flex", Flex: 1, MinWidth: 20},
		{Title: "fixed", Width: 10},
	}, 34)

	assert.GreaterOrEqual(t, widths[0], 20)
}

// A terminal too narrow for every column drops whole ones rather than
// squeezing them all into something unreadable.
func TestResolveWidths_DropsByPriority(t *testing.T) {
	columns := []Column{
		{Title: "keep", Flex: 1, MinWidth: 20},
		{Title: "important", Width: 10},
		{Title: "optional", Width: 10, Priority: 1},
		{Title: "expendable", Width: 10, Priority: 2},
	}

	widths := resolveWidths(columns, 40)

	assert.Zero(t, widths[3], "the most expendable column should go first")
	assert.Positive(t, widths[0])
	assert.Positive(t, widths[1])
}

func TestResolveWidths_NeverDropsPriorityZero(t *testing.T) {
	columns := []Column{
		{Title: "a", Width: 30},
		{Title: "b", Width: 30},
	}

	widths := resolveWidths(columns, 10)

	assert.Positive(t, widths[0])
	assert.Positive(t, widths[1])
}

func TestResolveWidths_Degenerate(t *testing.T) {
	assert.Empty(t, resolveWidths(nil, 100))
	assert.Equal(t, []int{0}, resolveWidths([]Column{{Title: "a", Width: 5}}, 0))
	assert.Equal(t, []int{0}, resolveWidths([]Column{{Title: "a", Width: 5}}, -10))
}

// Whatever the width, the columns and gaps add up to it exactly, or the table
// is narrower or wider than the region it was given.
func TestResolveWidths_FillsTheWidth(t *testing.T) {
	columns := []Column{
		{Title: "flex", Flex: 1, MinWidth: 10},
		{Title: "num", Width: 8, Align: AlignRight},
		{Title: "tags", Width: 12, Priority: 1},
	}

	for total := 40; total <= 200; total++ {
		widths := resolveWidths(columns, total)

		var kept int
		for _, w := range widths {
			if w > 0 {
				kept++
			}
		}

		assert.Equal(t, total, sum(widths)+(kept-1)*gap, "total=%d widths=%v", total, widths)
	}
}
