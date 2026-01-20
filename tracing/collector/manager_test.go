package collector

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	skprtime "github.com/skpr/compass/tracing/collector/time"
	"github.com/skpr/compass/tracing/trace"
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

	now := skprtime.NewMock(time.Now())

	manager, err := NewManager(logger, sink, Options{
		Expire: time.Second,
	}, now)
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
			Type:      EventRequestInit,
			RequestId: toUint8(requestID),
			Timestamp: uint64(100_000_000),
		},
		{
			Type:         EventFunction,
			RequestId:    toUint8(requestID),
			FunctionName: toUint8("Foo::bar"),
			Timestamp:    uint64(270_000_000),
			Elapsed:      uint64(120_000_000),
		},
		{
			Type:         EventFunction,
			RequestId:    toUint8(requestID),
			FunctionName: toUint8("Skpr::rocks"),
			Timestamp:    uint64(350_000_000),
			Elapsed:      uint64(80_000_000),
		},
		{
			Type:         EventFunction,
			RequestId:    toUint8(requestID),
			FunctionName: toUint8("Baz::boo"),
			Timestamp:    uint64(390_000_000),
			Elapsed:      uint64(40_000_000),
		},
		{
			Type:      EventRequestShutdown,
			RequestId: toUint8(requestID),
			Timestamp: uint64(390_000_000),
		},
	}

	for _, event := range events {
		err := manager.Handle(context.TODO(), event)
		assert.NoError(t, err)
	}

	// Check the profile that landed.
	assert.Equal(t, []trace.Trace{
		{
			Metadata: trace.Metadata{
				RequestID:      "123456789",
				StartTime:      now.Now(),
				MonotonicStart: 100_000_000,
				MonotonicEnd:   390_000_000,
			},
			FunctionCalls: []trace.FunctionCall{
				{
					Name:           "Foo::bar",
					MonotonicStart: 150_000_000,
					MonotonicEnd:   270_000_000,
					Elapsed:        120_000_000,
				},
				{
					Name:           "Skpr::rocks",
					MonotonicStart: 270_000_000,
					MonotonicEnd:   350_000_000,
					Elapsed:        80_000_000,
				},
				{
					Name:           "Baz::boo",
					MonotonicStart: 350_000_000,
					MonotonicEnd:   390_000_000,
					Elapsed:        40_000_000,
				},
			},
		},
	}, sink.Traces)
}
