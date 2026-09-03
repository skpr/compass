package cli

// bpfEvent is the normalized event shape retained for direct handler tests.
// Ring-buffer records use the compact generated type-specific structs instead.
type bpfEvent struct {
	Type         uint8
	Pid          uint64
	Command      [101]uint8
	FunctionName [101]uint8
	Timestamp    uint64
	Elapsed      uint64
	Memory       uint64
}
