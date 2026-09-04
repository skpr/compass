package main

import (
	"net/http"
	"net/http/httptest"
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

func TestRequireToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name   string
		token  string
		header string
		status int
	}{
		{name: "no token configured passes through", token: "", header: "", status: http.StatusOK},
		{name: "correct token reaches the handler", token: "s3cret", header: "s3cret", status: http.StatusOK},
		{name: "wrong token is rejected", token: "s3cret", header: "nope", status: http.StatusUnauthorized},
		{name: "missing token is rejected", token: "s3cret", header: "", status: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if tt.header != "" {
				req.Header.Set(HeaderToken, tt.header)
			}

			rec := httptest.NewRecorder()
			requireToken(tt.token, next).ServeHTTP(rec, req)

			assert.Equal(t, tt.status, rec.Code)
		})
	}
}
