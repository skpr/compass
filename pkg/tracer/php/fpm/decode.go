package fpm

import (
	"encoding/binary"

	"github.com/skpr/compass/pkg/tracer/ingest"
)

// Layout of the generated function record, struct function_event in
// program.bpf.c, as the C compiler lays it out:
//
//	offset  size  field
//	     0     1  type
//	     1   101  request_id
//	   102   101  function_name
//	   203     5  padding to the alignment of the fields below
//	   208     8  timestamp
//	   216     8  elapsed
//	   224     8  memory
const functionEventSize = 232

// decodeFunctionEvent reads one function record at the offsets above.
//
// Function calls are the only event which arrives more than a handful of times
// per request — a Drupal request can produce more than a million of them — so
// this is the one record whose decode cost sets the rate the collector can
// drain its ring buffer at. Walking the struct with reflection, which is what
// ingest.DecodeExact does, costs about 1.6µs per record against about 8ns
// here, and allocates. See docs/scaling.md.
//
// The offsets are written out rather than derived, so what keeps them honest is
// the test asserting that this agrees with the reflection decode, byte for
// byte, over random samples. That fails if the generated layout ever moves.
func decodeFunctionEvent(rawSample []byte) (bpfFunctionEvent, error) {
	var event bpfFunctionEvent

	if err := ingest.CheckSize(rawSample, functionEventSize); err != nil {
		return event, err
	}

	event.Type = rawSample[0]
	copy(event.RequestId[:], rawSample[1:102])
	copy(event.FunctionName[:], rawSample[102:203])
	event.Timestamp = binary.LittleEndian.Uint64(rawSample[208:216])
	event.Elapsed = binary.LittleEndian.Uint64(rawSample[216:224])
	event.Memory = binary.LittleEndian.Uint64(rawSample[224:232])

	return event, nil
}
