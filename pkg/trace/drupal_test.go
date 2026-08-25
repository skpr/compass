package trace

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The sidecar streams traces to the CLI as JSON, and the two are versioned and
// deployed independently. Drupal data has to be additive in both directions: a
// CLI which predates it ignores the field, and a CLI which knows about it has
// to cope with a sidecar which does not send one.
func TestTrace_JSON_DrupalIsOmittedWhenAbsent(t *testing.T) {
	encoded, err := json.Marshal(Trace{})
	require.NoError(t, err)

	assert.NotContains(t, string(encoded), "drupal")
}

func TestTrace_JSON_DecodesWithoutDrupal(t *testing.T) {
	var decoded Trace

	require.NoError(t, json.Unmarshal([]byte(`{"metadata":{"id":"req-1"},"functionCalls":[]}`), &decoded))

	assert.Equal(t, "req-1", decoded.Metadata.ID)
	assert.Nil(t, decoded.Drupal)
}

func TestTrace_JSON_RoundTripsDrupal(t *testing.T) {
	original := Trace{
		Metadata: Metadata{ID: "req-1"},
		Drupal: &Drupal{
			CacheEvents: []CacheEvent{
				{
					Origin:     CacheOriginObject,
					Caller:     `Drupal\node\NodeViewBuilder::build`,
					ObjectType: `Drupal\node\Entity\Node`,
					MaxAge:     0,
					Tags:       []string{"node:1"},
					Contexts:   []string{"user.roles"},
					StartTime:  1500,
					Calls:      3,
				},
			},
			CacheEventsDropped: 2,
		},
	}

	encoded, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Trace

	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, original, decoded)
}
