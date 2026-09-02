package app

import (
	"fmt"
	"testing"

	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/trace"
)

// BenchmarkSearchSetRowsDrupal exercises the configured maximum search list
// with the configured maximum number of distinct Drupal cache events per trace.
func BenchmarkSearchSetRowsDrupal(b *testing.B) {
	const (
		traces      = 500
		cacheEvents = 250
	)

	metadata := make([]trace.CacheEvent, 0, cacheEvents)
	for i := range cacheEvents {
		maxAge := int64(3600)
		if i == cacheEvents-1 {
			// Last makes the lightweight predicate take its worst-case path.
			maxAge = 0
		}

		metadata = append(metadata, trace.CacheEvent{
			Caller:   fmt.Sprintf("Drupal\\module\\Service%d::render", i),
			MaxAge:   maxAge,
			Calls:    1,
			Tags:     []string{fmt.Sprintf("node:%d", i)},
			Contexts: []string{fmt.Sprintf("context:%d", i)},
		})
	}

	m := NewModel("", traces, DefaultMaxLogs)
	m.Init()
	for i := range traces {
		m.traces.append(events.Trace{Trace: trace.Trace{
			Metadata: trace.Metadata{
				ID:      fmt.Sprintf("request-%d", i),
				Runtime: trace.RuntimePHP,
				Source:  trace.SourceHTTP,
				HTTP:    trace.MetadataHTTP{Method: "GET", URI: fmt.Sprintf("/node/%d", i)},
			},
			Drupal: &trace.Drupal{CacheEvents: metadata},
		}})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		m.searchSetRows()
	}
}
