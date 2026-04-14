package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/skpr/compass/pkg/trace"
)

func TestTrace_Title_HTTP(t *testing.T) {
	tr := Trace{
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				Source:    trace.SourceHTTP,
				StartTime: 0,
				EndTime:   2_000_000, // 2ms
				HTTP: trace.MetadataHTTP{
					Method: "GET",
					URI:    "/api/test",
				},
			},
			ResourceUtilisation: trace.ResourceUtilisation{
				MaxMemory: 500,
			},
		},
	}

	title := tr.Title()
	assert.Contains(t, title, "2ms")
	assert.Contains(t, title, "500 B")
	assert.Contains(t, title, "GET")
	assert.Contains(t, title, "/api/test")
}

func TestTrace_Title_CLI(t *testing.T) {
	tr := Trace{
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				Source:    trace.SourceCLI,
				StartTime: 0,
				EndTime:   5_000_000, // 5ms
				CLI: trace.MetadataCLI{
					Command: "drush cr",
				},
			},
			ResourceUtilisation: trace.ResourceUtilisation{
				MaxMemory: 2048,
			},
		},
	}

	title := tr.Title()
	assert.Contains(t, title, "5ms")
	assert.Contains(t, title, "2.0 KB")
	assert.Contains(t, title, "drush cr")
}

func TestTrace_Title_Unknown(t *testing.T) {
	tr := Trace{
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				Source:    trace.Source("other"),
				StartTime: 0,
				EndTime:   1_000_000,
			},
			ResourceUtilisation: trace.ResourceUtilisation{
				MaxMemory: 100,
			},
		},
	}

	title := tr.Title()
	assert.Contains(t, title, "UNKNOWN")
}

func TestTrace_Description(t *testing.T) {
	now := time.Now()
	tr := Trace{
		IngestionTime: now,
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				Source: trace.SourceHTTP,
				ID:     "req-123",
			},
			FunctionCalls: []trace.FunctionCall{
				{Name: "foo"},
				{Name: "bar"},
			},
		},
	}

	desc := tr.Description()
	assert.Contains(t, desc, "id=req-123")
	assert.Contains(t, desc, "2 calls")
	assert.Contains(t, desc, "just now")
}

func TestTrace_Description_OlderTrace(t *testing.T) {
	tr := Trace{
		IngestionTime: time.Now().Add(-5 * time.Minute),
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				Source: trace.SourceHTTP,
			},
			FunctionCalls: []trace.FunctionCall{
				{Name: "foo"},
			},
		},
	}

	desc := tr.Description()
	assert.Contains(t, desc, "1 calls")
	assert.Contains(t, desc, "5m ago")
}

func TestTrace_FilterValue(t *testing.T) {
	tr := Trace{
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				Source:    trace.SourceHTTP,
				Runtime:   trace.RuntimePHP,
				StartTime: 0,
				EndTime:   1_000_000,
				HTTP: trace.MetadataHTTP{
					Method: "POST",
					URI:    "/submit",
				},
			},
			ResourceUtilisation: trace.ResourceUtilisation{
				MaxMemory: 100,
			},
		},
	}

	filterValue := tr.FilterValue()
	assert.Contains(t, filterValue, "php")
	assert.Contains(t, filterValue, "http")
	assert.Contains(t, filterValue, "POST")
	assert.Contains(t, filterValue, "/submit")
}

func TestRelativeTime_JustNow(t *testing.T) {
	assert.Equal(t, "just now", relativeTime(time.Now()))
}

func TestRelativeTime_Seconds(t *testing.T) {
	assert.Equal(t, "30s ago", relativeTime(time.Now().Add(-30*time.Second)))
}

func TestRelativeTime_Minutes(t *testing.T) {
	assert.Equal(t, "3m ago", relativeTime(time.Now().Add(-3*time.Minute)))
}

func TestRelativeTime_Hours(t *testing.T) {
	assert.Equal(t, "2h ago", relativeTime(time.Now().Add(-2*time.Hour)))
}

func TestRelativeTime_Zero(t *testing.T) {
	assert.Equal(t, "just now", relativeTime(time.Time{}))
}

func TestTrace_Title_DurationContainsMs(t *testing.T) {
	tr := Trace{
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				Source:    trace.SourceHTTP,
				Runtime:   trace.RuntimePHP,
				StartTime: 0,
				EndTime:   600_000_000, // 600ms - slow
				HTTP: trace.MetadataHTTP{
					Method: "GET",
					URI:    "/slow",
				},
			},
			ResourceUtilisation: trace.ResourceUtilisation{
				MaxMemory: 1024,
			},
		},
	}

	title := tr.Title()
	assert.Contains(t, title, "600ms")
	assert.Contains(t, title, "GET")
	assert.Contains(t, title, "/slow")
}

func TestTrace_DurationBar_Fast(t *testing.T) {
	tr := Trace{
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				StartTime: 0,
				EndTime:   50_000_000, // 50ms
			},
		},
	}

	bar := tr.durationBar()
	assert.Contains(t, bar, "░")
	assert.Contains(t, bar, "█")
}

func TestTrace_DurationBar_Full(t *testing.T) {
	tr := Trace{
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				StartTime: 0,
				EndTime:   1_500_000_000, // 1500ms - over max
			},
		},
	}

	bar := tr.durationBar()
	assert.Contains(t, bar, "████████")
	assert.NotContains(t, bar, "░")
}

func TestTrace_DurationBar_Zero(t *testing.T) {
	tr := Trace{
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				StartTime: 0,
				EndTime:   0,
			},
		},
	}

	bar := tr.durationBar()
	assert.Contains(t, bar, "░░░░░░░░")
}

func TestTrace_DurationBar_InTitle(t *testing.T) {
	tr := Trace{
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				Source:    trace.SourceHTTP,
				Runtime:   trace.RuntimePHP,
				StartTime: 0,
				EndTime:   500_000_000, // 500ms
				HTTP: trace.MetadataHTTP{
					Method: "GET",
					URI:    "/test",
				},
			},
			ResourceUtilisation: trace.ResourceUtilisation{
				MaxMemory: 1024,
			},
		},
	}

	title := tr.Title()
	assert.Contains(t, title, "█")
	assert.Contains(t, title, "500ms")
	assert.Contains(t, title, "GET")
}

func TestFormatBytes_Zero(t *testing.T) {
	assert.Equal(t, "0 B", formatBytes(0))
}

func TestFormatBytes_Bytes(t *testing.T) {
	assert.Equal(t, "500 B", formatBytes(500))
}

func TestFormatBytes_Kilobytes(t *testing.T) {
	assert.Equal(t, "1.0 KB", formatBytes(1024))
}

func TestFormatBytes_Megabytes(t *testing.T) {
	assert.Equal(t, "1.0 MB", formatBytes(1024*1024))
}

func TestFormatBytes_Gigabytes(t *testing.T) {
	assert.Equal(t, "1.0 GB", formatBytes(1024*1024*1024))
}
