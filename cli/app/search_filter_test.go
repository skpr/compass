package app

import (
	"testing"

	"github.com/skpr/compass/trace"
	"github.com/stretchr/testify/assert"
)

func TestSearchFilter(t *testing.T) {
	model := &Model{
		breakdownScroll: 3,
	}

	// Not query provided.
	model.traces = Traces{
		All: []trace.Trace{
			{
				Metadata: trace.Metadata{
					RequestID: "aaabbbccc",
				},
			},
			{
				Metadata: trace.Metadata{
					RequestID: "xxxyyyzzz",
				},
			},
		},
	}
	model.SearchFilter()
	assert.Equal(t, 2, len(model.traces.Filtered))

	// Query provided.
	model.traces = Traces{
		All: []trace.Trace{
			{
				Metadata: trace.Metadata{
					RequestID: "aaabbbccc",
					URI:       "/foo",
				},
			},
			{
				Metadata: trace.Metadata{
					RequestID: "xxxyyyzzz",
					URI:       "/bar",
				},
			},
		},
	}
	model.searchQuery = "foo"
	model.SearchFilter()
	assert.Equal(t, 1, len(model.traces.Filtered))
}
