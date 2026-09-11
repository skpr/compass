package trace

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrace_JSON_CallsDroppedIsOmittedWhenZero(t *testing.T) {
	encoded, err := json.Marshal(Trace{})
	require.NoError(t, err)

	assert.NotContains(t, string(encoded), "callsDropped")
}

func TestTrace_JSON_DecodesWithoutCallsDropped(t *testing.T) {
	var decoded Trace

	require.NoError(t, json.Unmarshal([]byte(`{"spans":[]}`), &decoded))
	assert.Zero(t, decoded.CallsDropped)
}

func TestTrace_JSON_RoundTripsSpans(t *testing.T) {
	original := Trace{
		Spans: []Span{{
			Name:    "Drupal\\Core\\Entity::loadMultiple",
			Offset:  2 * time.Millisecond,
			Elapsed: 900 * time.Microsecond,
			Total:   1300 * time.Microsecond,
			Calls:   3,
			Memory:  1 << 20,
		}},
		Calls:        41,
		CallsDropped: 37,
	}

	encoded, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Trace
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, original, decoded)
}

func TestSpan_DurationShare(t *testing.T) {
	span := Span{Elapsed: 400 * time.Millisecond}

	assert.InDelta(t, 0.4, span.DurationShare(time.Second), 0.0001)
	assert.Zero(t, span.DurationShare(0))
}
