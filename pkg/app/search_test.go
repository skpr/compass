package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// An X-Ray trace id opens with "Root=1-" and an epoch every nearby request
// shares, so the list has to show the random part of it instead.
func TestShortID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"hex", "58bb2c6e56c13ce04c1cb9a87083d735", "58bb2c6e"},
		{"short", "req-1", "req-1"},
		{"xray", "Root=1-6ab498e9-2b18ff2f0e49bc425910bfc6", "2b18ff2f"},
		{"xray with fields", "Root=1-6ab498e9-2b18ff2f0e49bc425910bfc6;Parent=53995c3f42cd8ad8;Sampled=1", "2b18ff2f"},
		{"xray after self", "Self=1-67891234-12456789abcdef012345678;Root=1-6ab498e9-2b18ff2f0e49bc425910bfc6", "2b18ff2f"},
		{"malformed root", "Root=abc", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shortID(tt.id))
		})
	}
}
