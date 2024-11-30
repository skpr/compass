package collector

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/skpr/compass/trace"
)

// TestSync for storing complete profile data for our tests.
type TestSync struct {
	Traces []trace.Trace
}

// Initialize the sink.
func (t *TestSync) Initialize() error {
	return nil
}

// ProcessProfile which has been collected.
func (t *TestSync) ProcessTrace(trace trace.Trace) error {
	t.Traces = append(t.Traces, trace)
	return nil
}

func TestHandleRequestShutdown(t *testing.T) {
	// Store the logs for later.
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	// Sink for reviewing the compiled profile.
	sink := &TestSync{}

	manager, err := NewManager(logger, sink, Options{
		Expire: time.Second,
	})
	assert.NoError(t, err)

	toUint8 := func(val string) [101]uint8 {
		var arr [101]uint8

		if len(val) > 101 {
			panic("string is too long for a [101]uint8 array")
		}

		for i, char := range val {
			arr[i] = uint8(char)
		}

		return arr
	}

	requestID := "123456789"

	events := []bpfEvent{
		{
			Type:         toUint8(EventFunction),
			RequestId:    toUint8(requestID),
			ClassName:    toUint8("Foo"),
			FunctionName: toUint8("bar"),
			StartTime:    uint64(3000000),
			EndTime:      uint64(15000000),
		},
		{
			Type:         toUint8(EventFunction),
			RequestId:    toUint8(requestID),
			ClassName:    toUint8("Skpr"),
			FunctionName: toUint8("rocks"),
			StartTime:    uint64(5000000),
			EndTime:      uint64(13000000),
		},
		{
			Type:         toUint8(EventFunction),
			RequestId:    toUint8(requestID),
			ClassName:    toUint8("Baz"),
			FunctionName: toUint8("boo"),
			StartTime:    uint64(6000000),
			EndTime:      uint64(10000000),
		},
		{
			Type:      toUint8(EventRequestShutdown),
			RequestId: toUint8(requestID),
		},
	}

	for _, event := range events {
		err := manager.Handle(event)
		assert.NoError(t, err)
	}

	err = manager.Shutdown()
	assert.NoError(t, err)

	// Check the profile that landed.
	assert.Equal(t, []trace.Trace{
		{
			RequestID:     "123456789",
			StartTime:     3000000,
			EndTime:       15000000,
			ExecutionTime: 12000,
			FunctionCalls: []trace.FunctionCall{
				{
					Name:      "Foo::bar",
					StartTime: 3000000,
					EndTime:   15000000,
				},
				{
					Name:      "Skpr::rocks",
					StartTime: 5000000,
					EndTime:   13000000,
				},
				{
					Name:      "Baz::boo",
					StartTime: 6000000,
					EndTime:   10000000,
				},
			},
		},
	}, sink.Traces)
}
