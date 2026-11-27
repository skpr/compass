// Package export writes traces to files, so they can be read by something
// other than the terminal UI: jq, a diff, a ticket, or an agent like Claude
// Code which is debugging the request.
//
// The interface exports the trace being looked at. Which traces are worth
// exporting is the reader's call, made with the same search and filter they
// already use to find them.
package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/skpr/compass/pkg/trace"
)

// DefaultDir is where traces are written when no directory has been
// configured.
const DefaultDir = "compass-traces"

// unsafe matches what is not kept from a request ID when it is used in a file
// name. The ID comes from a request header, so it is whatever the client sent.
var unsafe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// maxIDLength in a file name. The full ID is in the file regardless.
const maxIDLength = 64

// Write a trace to a file in dir, creating dir if needed, and return the path
// it was written to.
//
// The file is the trace exactly as the sidecar streams it, as one indented
// JSON document, so anything which reads the stream reads an export.
func Write(dir string, t trace.Trace) (string, error) {
	if dir == "" {
		dir = DefaultDir
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	encoded, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode trace: %w", err)
	}

	path := filepath.Join(dir, FileName(t))

	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("failed to write trace: %w", err)
	}

	return path, nil
}

// FileName for a trace: when it started, then what identifies it.
//
// Starting with the time means a directory of exports lists in the order the
// requests ran. Nanoseconds are included because two requests in the same
// second are common, and without them the second would replace the first.
func FileName(t trace.Trace) string {
	name := t.Metadata.StartTime.UTC().Format("20060102T150405.000000000Z")

	id := ""
	if t.Metadata.Identified() {
		id = strings.Trim(unsafe.ReplaceAllString(t.Metadata.ID, "_"), "_.")
	}

	if len(id) > maxIDLength {
		id = id[:maxIDLength]
	}

	if id != "" {
		name += "-" + id
	}

	return name + ".json"
}
