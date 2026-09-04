package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthorized(t *testing.T) {
	tests := []struct {
		name string
		want string
		got  string
		ok   bool
	}{
		{name: "no token configured allows anything", want: "", got: "", ok: true},
		{name: "no token configured ignores a presented one", want: "", got: "whatever", ok: true},
		{name: "correct token", want: "s3cret", got: "s3cret", ok: true},
		{name: "wrong token of the same length", want: "s3cret", got: "s3crat", ok: false},
		{name: "wrong token of a different length", want: "s3cret", got: "s3", ok: false},
		{name: "missing token when one is required", want: "s3cret", got: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.ok, authorized(tt.want, tt.got))
		})
	}
}
