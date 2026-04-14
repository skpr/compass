package span

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFractionDuration_Half(t *testing.T) {
	result := FractionDuration(50*time.Millisecond, 100*time.Millisecond)
	assert.InDelta(t, 0.5, result, 0.001)
}

func TestFractionDuration_Full(t *testing.T) {
	result := FractionDuration(100*time.Millisecond, 100*time.Millisecond)
	assert.InDelta(t, 1.0, result, 0.001)
}

func TestFractionDuration_Zero(t *testing.T) {
	result := FractionDuration(0, 100*time.Millisecond)
	assert.InDelta(t, 0.0, result, 0.001)
}

func TestFractionDuration_ZeroTotal(t *testing.T) {
	result := FractionDuration(50*time.Millisecond, 0)
	assert.InDelta(t, 0.0, result, 0.001)
}

func TestToPositiveInt_Positive(t *testing.T) {
	assert.Equal(t, 3, toPositiveInt(3.7))
}

func TestToPositiveInt_Negative(t *testing.T) {
	assert.Equal(t, 0, toPositiveInt(-1.5))
}

func TestToPositiveInt_Zero(t *testing.T) {
	assert.Equal(t, 0, toPositiveInt(0.0))
}

func TestTidyFill_Exact(t *testing.T) {
	// total == want → fill unchanged.
	assert.Equal(t, 10, tidyFill(50, 50, 10))
}

func TestTidyFill_TotalGreater(t *testing.T) {
	// total (52) > want (50) → fill reduced by 2.
	assert.Equal(t, 8, tidyFill(52, 50, 10))
}

func TestTidyFill_WantGreater(t *testing.T) {
	// want (50) > total (48) → fill increased by 2.
	assert.Equal(t, 12, tidyFill(48, 50, 10))
}

func TestColorForFill_Red(t *testing.T) {
	result := colorForFill(0.80, "##")
	// Should contain the block text (colored).
	assert.Contains(t, result, "##")
}

func TestColorForFill_Orange(t *testing.T) {
	result := colorForFill(0.60, "##")
	assert.Contains(t, result, "##")
}

func TestColorForFill_Yellow(t *testing.T) {
	result := colorForFill(0.30, "##")
	assert.Contains(t, result, "##")
}

func TestColorForFill_Blue(t *testing.T) {
	result := colorForFill(0.10, "##")
	assert.Contains(t, result, "##")
}

func TestComponent_Render(t *testing.T) {
	c := New(100*time.Millisecond, 50)

	s := Span{
		Start:    10 * time.Millisecond,
		Duration: 20 * time.Millisecond,
	}

	result := c.Render(s)

	// Should contain the duration in milliseconds.
	assert.Contains(t, result, "20ms")
}

func TestComponent_Render_ZeroDuration(t *testing.T) {
	c := New(100*time.Millisecond, 50)

	s := Span{
		Start:    0,
		Duration: 0,
	}

	result := c.Render(s)
	assert.Contains(t, result, "0ms")
}
