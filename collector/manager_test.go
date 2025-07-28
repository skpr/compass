package collector

import (
	"bytes"
	"context"
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
func (t *TestSync) ProcessTrace(_ context.Context, trace trace.Trace) error {
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
			Type:      1,
			RequestId: toUint8(requestID),
			Timestamp: uint64(3000000),
		},
		{
			Type:         0,
			RequestId:    toUint8(requestID),
			FunctionName: toUint8("Foo::bar"),
			Timestamp:    uint64(15000000),
			Elapsed:      uint64(12000000),
		},
		{
			Type:         0,
			RequestId:    toUint8(requestID),
			FunctionName: toUint8("Skpr::rocks"),
			Timestamp:    uint64(13000000),
			Elapsed:      uint64(8000000),
		},
		{
			Type:         0,
			RequestId:    toUint8(requestID),
			FunctionName: toUint8("Baz::boo"),
			Timestamp:    uint64(10000000),
			Elapsed:      uint64(4000000),
		},
		{
			Type:      2,
			RequestId: toUint8(requestID),
			Timestamp: uint64(15000000),
		},
	}

	for _, event := range events {
		err := manager.Handle(event)
		assert.NoError(t, err)
	}

	// Check the profile that landed.
	assert.Equal(t, []trace.Trace{
		{
			Metadata: trace.Metadata{
				RequestID: "123456789",
				StartTime: 3000000,
				EndTime:   15000000,
			},
			FunctionCalls: []trace.FunctionCall{
				{
					Name:      "Foo::bar",
					StartTime: 3000000,
					Elapsed:   12000000,
				},
				{
					Name:      "Skpr::rocks",
					StartTime: 5000000,
					Elapsed:   8000000,
				},
				{
					Name:      "Baz::boo",
					StartTime: 6000000,
					Elapsed:   4000000,
				},
			},
		},
	}, sink.Traces)
}
