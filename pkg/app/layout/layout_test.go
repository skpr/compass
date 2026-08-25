package layout

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The property that matters: whatever the terminal size and whatever the screen
// asks for, no region is ever negative. Every layout bug this package exists to
// prevent showed up first as a negative height handed to a component.
func TestCompute_NeverNegative(t *testing.T) {
	sizes := []struct{ width, height int }{
		{200, 60}, {120, 40}, {80, 24}, {60, 12}, {60, 11},
		{59, 40}, {40, 10}, {80, 8}, {1, 1}, {0, 0}, {-5, -5},
	}

	for _, opts := range []Options{{}, {DetailHeight: 4}, {Filter: true}, {InspectHeight: 5}, {DetailHeight: 4, Filter: true, InspectHeight: 5}} {
		for _, size := range sizes {
			regions := Compute(size.width, size.height, opts)

			assert.GreaterOrEqual(t, regions.Banner, 0)
			assert.GreaterOrEqual(t, regions.Menu, 0)
			assert.GreaterOrEqual(t, regions.Detail, 0)
			assert.GreaterOrEqual(t, regions.Content, 0)
			assert.GreaterOrEqual(t, regions.Footer, 0)
		}
	}
}

// The regions have to add up, or the interface renders taller or shorter than
// the terminal and the whole screen scrolls.
func TestCompute_RegionsSumToTheTerminalHeight(t *testing.T) {
	for _, opts := range []Options{{}, {DetailHeight: 4}, {Filter: true}, {InspectHeight: 5}, {DetailHeight: 4, Filter: true, InspectHeight: 5}} {
		for height := MinHeight; height <= 60; height++ {
			regions := Compute(120, height, opts)
			if regions.TooSmall {
				continue
			}

			assert.Equal(t, height, regions.Total(), "height=%d opts=%+v", height, opts)
		}
	}
}

func TestCompute_TooSmall(t *testing.T) {
	assert.True(t, Compute(MinWidth-1, 40, Options{}).TooSmall)
	assert.True(t, Compute(120, MinHeight-1, Options{}).TooSmall)
	assert.False(t, Compute(MinWidth, MinHeight, Options{}).TooSmall)
}

func TestCompute_TooSmallZeroesEverything(t *testing.T) {
	regions := Compute(10, 4, Options{DetailHeight: 4, Filter: true, InspectHeight: 5})

	require.True(t, regions.TooSmall)
	assert.Zero(t, regions.Banner)
	assert.Zero(t, regions.Menu)
	assert.Zero(t, regions.Detail)
	assert.Zero(t, regions.Inspect)
	assert.Zero(t, regions.Content)
	assert.Zero(t, regions.Footer)
}

// On a short terminal the panels give way before the rows do. A trace's
// function list is worth more than a summary of the trace you already opened.
func TestCompute_DetailGivesWayBeforeContent(t *testing.T) {
	regions := Compute(120, MinHeight, Options{DetailHeight: 4})

	require.False(t, regions.TooSmall)
	assert.Zero(t, regions.Detail)
	assert.GreaterOrEqual(t, regions.Content, MinContentHeight)
}

// And the panel describing one row goes before the one describing the trace: a
// screen with no room for rows has no row to describe.
func TestCompute_InspectGivesWayFirst(t *testing.T) {
	regions := Compute(120, MinHeight+4, Options{DetailHeight: 4, InspectHeight: 5})

	require.False(t, regions.TooSmall)
	assert.Zero(t, regions.Inspect)
	assert.Equal(t, 4, regions.Detail)
}

// Whenever the interface is drawn at all, there is a usable content region.
func TestCompute_ContentIsAlwaysUsableWhenDrawn(t *testing.T) {
	for _, opts := range []Options{{}, {DetailHeight: 4}, {Filter: true}, {InspectHeight: 5}, {DetailHeight: 4, Filter: true, InspectHeight: 5}} {
		for height := 0; height <= 60; height++ {
			regions := Compute(120, height, opts)
			if regions.TooSmall {
				continue
			}

			assert.GreaterOrEqual(t, regions.Content, MinContentHeight, "height=%d opts=%+v", height, opts)
		}
	}
}

func TestCompute_RoomyTerminalKeepsEverything(t *testing.T) {
	regions := Compute(200, 60, Options{DetailHeight: 4, Filter: true, InspectHeight: 5})

	require.False(t, regions.TooSmall)
	assert.Equal(t, BannerHeight, regions.Banner)
	assert.Equal(t, MenuHeight, regions.Menu)
	assert.Equal(t, 4, regions.Detail)
	assert.Equal(t, FilterHeight, regions.Filter)
	assert.Equal(t, 5, regions.Inspect)
	assert.Equal(t, FooterHeight, regions.Footer)
	assert.Equal(t, 60-BannerHeight-MenuHeight-4-FilterHeight-5-FooterHeight, regions.Content)
}

// Once every strip fits, growing the terminal grows the rows and nothing else.
//
// Below that point content is deliberately not monotonic: at the size where a
// strip becomes affordable it comes back, and it costs a row to do so. That is
// the right trade — the reader gets the summary they asked for — but it means
// monotonicity only holds above the threshold where nothing is being dropped.
func TestCompute_ContentGrowsMonotonicallyOnceEverythingFits(t *testing.T) {
	full := MinHeight + 4 + 5

	previous := 0

	for height := full; height <= 100; height++ {
		regions := Compute(120, height, Options{DetailHeight: 4, InspectHeight: 5})
		require.False(t, regions.TooSmall)

		assert.Equal(t, 4, regions.Detail, "height=%d", height)
		assert.Equal(t, 5, regions.Inspect, "height=%d", height)
		assert.Greater(t, regions.Content, previous, "content did not grow at height %d", height)

		previous = regions.Content
	}
}

// The blocks are however many rows their contents need, so the layout takes
// heights rather than flags. Nothing to show means no region.
func TestCompute_HeightsAreGiven(t *testing.T) {
	for _, height := range []int{0, 1, 4, 7} {
		regions := Compute(120, 60, Options{DetailHeight: height, InspectHeight: height})

		assert.Equal(t, height, regions.Detail, "height=%d", height)
		assert.Equal(t, height, regions.Inspect, "height=%d", height)
	}

	// A negative is not a region which pushes everything else up by a row.
	regions := Compute(120, 60, Options{DetailHeight: -3, InspectHeight: -3})

	assert.Zero(t, regions.Detail)
	assert.Zero(t, regions.Inspect)
}
