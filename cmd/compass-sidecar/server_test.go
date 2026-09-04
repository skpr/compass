package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer_SetsSlowClientTimeouts(t *testing.T) {
	server := newServer(":28624", http.NewServeMux())

	assert.Equal(t, ":28624", server.Addr)
	assert.Equal(t, serverReadHeaderTimeout, server.ReadHeaderTimeout)
	assert.Equal(t, serverIdleTimeout, server.IdleTimeout)
}

func TestNewServer_DoesNotCapTheStream(t *testing.T) {
	server := newServer(":28624", http.NewServeMux())

	assert.Zero(t, server.WriteTimeout)
}

func TestNewServer_DisconnectsSlowHeaderClient(t *testing.T) {
	prev := serverReadHeaderTimeout
	serverReadHeaderTimeout = 150 * time.Millisecond
	t.Cleanup(func() { serverReadHeaderTimeout = prev })

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := newServer(listener.Addr().String(), http.NewServeMux())

	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	defer conn.Close()

	// A request line and one header, but no terminating blank line.
	_, err = fmt.Fprint(conn, "GET /metrics HTTP/1.1\r\nHost: localhost\r\n")
	require.NoError(t, err)

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))

	// The read completing (with EOF) rather than blocking is the property here.
	_, err = io.ReadAll(conn)
	assert.NoError(t, err, "server should close the connection, not leave it hanging")
}

func TestNewServer_ServesAfterHeaderTimeout(t *testing.T) {
	prev := serverReadHeaderTimeout
	serverReadHeaderTimeout = 150 * time.Millisecond
	t.Cleanup(func() { serverReadHeaderTimeout = prev })

	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	ts := httptest.NewUnstartedServer(mux)
	ts.Config = newServer("", mux)
	ts.Start()
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/ok")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}
