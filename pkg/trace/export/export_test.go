package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
)

var origin = time.Date(2026, 11, 25, 1, 2, 3, 456, time.UTC)

func request(id, method, uri string, duration time.Duration) trace.Trace {
	return trace.Trace{
		Metadata: trace.Metadata{
			Source:    trace.SourceHTTP,
			Runtime:   trace.RuntimePHP,
			ID:        id,
			HTTP:      trace.MetadataHTTP{Method: method, URI: uri},
			StartTime: origin,
			EndTime:   origin.Add(duration),
		},
	}
}

// An export is read by whatever reads the stream, so it has to decode back to
// the trace it was written from.
func TestWrite_RoundTrips(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested")

	original := request("req-1", "GET", "/node/1", time.Second)
	original.Spans = []trace.Span{{Name: `Drupal\Core\Entity::load`, Offset: time.Millisecond, Elapsed: 2 * time.Millisecond, Total: 3 * time.Millisecond, Calls: 2}}
	original.Calls = 2

	path, err := Write(dir, original)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "20261125T010203.000000456Z-req-1.json"), path)

	encoded, err := os.ReadFile(path)
	require.NoError(t, err)

	var decoded trace.Trace
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, original.Metadata.ID, decoded.Metadata.ID)
	assert.Equal(t, original.Spans, decoded.Spans)
	assert.True(t, original.Metadata.StartTime.Equal(decoded.Metadata.StartTime))
}

// The ID is whatever the client put in a header, so it cannot be trusted to
// stay inside the export directory.
func TestFileName_SanitisesID(t *testing.T) {
	name := FileName(request("../../etc/passwd", "GET", "/", 0))

	assert.Equal(t, "20261125T010203.000000456Z-etc_passwd.json", name)
}

func TestFileName_UnidentifiedHasNoID(t *testing.T) {
	assert.Equal(t, "20261125T010203.000000456Z.json", FileName(request(trace.IDUnknown, "GET", "/", 0)))
	assert.Equal(t, "20261125T010203.000000456Z.json", FileName(request("", "GET", "/", 0)))
}
