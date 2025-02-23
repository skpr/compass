package span

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name   string
		start  int64
		span   int64
		length int64
		expect string
	}{
		{
			name:   "Basic case",
			start:  2,
			span:   3,
			length: 10,
			expect: "│  ■■■     │",
		},
		{
			name:   "Span at the beginning",
			start:  0,
			span:   5,
			length: 10,
			expect: "│■■■■■     │",
		},
		{
			name:   "Span at the end",
			start:  7,
			span:   3,
			length: 10,
			expect: "│       ■■■│",
		},
		{
			name:   "Full length span",
			start:  0,
			span:   10,
			length: 10,
			expect: "│■■■■■■■■■■│",
		},
		{
			name:   "No span",
			start:  5,
			span:   0,
			length: 10,
			expect: "│          │",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Render(tc.start, tc.span, tc.length)
			assert.Equal(t, tc.expect, result)
		})
	}
}
