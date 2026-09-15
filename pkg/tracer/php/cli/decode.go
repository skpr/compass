package cli

import (
	"encoding/binary"

	"github.com/skpr/compass/pkg/tracer/ingest"
)

// Layout of the generated function record, struct function_event in
// program.bpf.c, as the C compiler lays it out. A CLI run is identified by its
// process id rather than a request id, so this record carries a pid where the
// FPM one carries a request id:
//
//	offset  size  field
//	     0     1  type
//	     1   101  function_name
//	   102     2  padding to the alignment of the fields below
//	   104     8  pid
//	   112     8  timestamp
//	   120     8  elapsed
//	   128     8  memory
const functionEventSize = 136

// decodeFunctionEvent reads one function record at the offsets above.
//
// Function calls are the only event which arrives more than a handful of times
// per run — a Drush command can produce a great many of them — so this is the
// one record whose decode cost sets the rate the collector can drain its ring
// buffer at. Walking the struct with reflection, which is what
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
	copy(event.FunctionName[:], rawSample[1:102])
	event.Pid = binary.LittleEndian.Uint64(rawSample[104:112])
	event.Timestamp = binary.LittleEndian.Uint64(rawSample[112:120])
	event.Elapsed = binary.LittleEndian.Uint64(rawSample[120:128])
	event.Memory = binary.LittleEndian.Uint64(rawSample[128:136])

	return event, nil
}
