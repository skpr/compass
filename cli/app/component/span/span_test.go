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

func TestToPositiveInt(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  int
	}{
		{
			name:  "Positive input",
			input: 5,
			want:  5,
		},
		{
			name:  "Zero input",
			input: 0,
			want:  0,
		},
		{
			name:  "Negative input",
			input: -3,
			want:  0,
		},
		{
			name:  "Large positive input",
			input: 1234567890,
			want:  1234567890,
		},
		{
			name:  "Large negative input",
			input: -9876543210,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toPositiveInt(tt.input)
			if got != tt.want {
				t.Errorf("toPositiveInt(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
