package trace

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrace_JSON_FunctionCallsDroppedIsOmittedWhenZero(t *testing.T) {
	encoded, err := json.Marshal(Trace{})
	require.NoError(t, err)

	assert.NotContains(t, string(encoded), "functionCallsDropped")
}

func TestTrace_JSON_DecodesWithoutFunctionCallsDropped(t *testing.T) {
	var decoded Trace

	require.NoError(t, json.Unmarshal([]byte(`{"functionCalls":[]}`), &decoded))
	assert.Zero(t, decoded.FunctionCallsDropped)
}

func TestTrace_JSON_RoundTripsFunctionCallsDropped(t *testing.T) {
	original := Trace{
		FunctionCalls:        []FunctionCall{{Name: "retained"}},
		FunctionCallsDropped: 37,
	}

	encoded, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Trace
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, original, decoded)
}
