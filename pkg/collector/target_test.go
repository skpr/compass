package collector

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTarget(t *testing.T) {
	const canonical = "12345678-1234-1234-1234-123456789abc"

	cases := map[string]struct {
		query   string
		want    Target
		wantErr error
	}{
		"dashed uid": {
			query: "uid=12345678-1234-1234-1234-123456789abc",
			want:  Target{UID: canonical},
		},
		"underscored uid normalises": {
			query: "uid=12345678_1234_1234_1234_123456789abc",
			want:  Target{UID: canonical},
		},
		"uppercase uid normalises": {
			query: "uid=12345678-1234-1234-1234-123456789ABC",
			want:  Target{UID: canonical},
		},
		"missing uid": {
			query:   "",
			wantErr: ErrMissingUID,
		},
		"empty uid": {
			query:   "uid=",
			wantErr: ErrMissingUID,
		},
		"malformed uid": {
			query:   "uid=not-a-pod",
			wantErr: ErrInvalidUID,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			values, err := url.ParseQuery(tc.query)
			require.NoError(t, err)

			got, err := ParseTarget(values)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTargetKey(t *testing.T) {
	// Two spellings of the same pod must produce the same routing key.
	a, err := ParseTarget(url.Values{"uid": {"12345678-1234-1234-1234-123456789abc"}})
	require.NoError(t, err)

	b, err := ParseTarget(url.Values{"uid": {"12345678_1234_1234_1234_123456789ABC"}})
	require.NoError(t, err)

	assert.Equal(t, a.Key(), b.Key())
}
