package usdt

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeDesc builds a well-formed note descriptor: three little-endian addresses
// followed by null-terminated provider, probe and args strings.
func makeDesc(location, base, semaphore uint64, provider, probe, args string) []byte {
	desc := make([]byte, 3*8)
	binary.LittleEndian.PutUint64(desc[0:8], location)
	binary.LittleEndian.PutUint64(desc[8:16], base)
	binary.LittleEndian.PutUint64(desc[16:24], semaphore)

	desc = append(desc, []byte(provider)...)
	desc = append(desc, 0)
	desc = append(desc, []byte(probe)...)
	desc = append(desc, 0)
	desc = append(desc, []byte(args)...)
	desc = append(desc, 0)

	return desc
}

func TestParseNoteDesc(t *testing.T) {
	desc := makeDesc(0x1000, 0x2000, 0x3000, "compass", "fpm_function", "-8@%rax -8@%rcx")

	d, err := parseNoteDesc(desc, 8, binary.LittleEndian)
	require.NoError(t, err)

	assert.Equal(t, uint64(0x1000), d.location)
	assert.Equal(t, uint64(0x2000), d.base)
	assert.Equal(t, uint64(0x3000), d.semaphore)
	assert.Equal(t, "compass", d.provider)
	assert.Equal(t, "fpm_function", d.probe)
	assert.Equal(t, "-8@%rax -8@%rcx", d.args)
}

func TestParseNoteDesc_NoArgs(t *testing.T) {
	desc := makeDesc(1, 2, 3, "compass", "canary", "")

	d, err := parseNoteDesc(desc, 8, binary.LittleEndian)
	require.NoError(t, err)

	assert.Equal(t, "canary", d.probe)
	assert.Empty(t, d.args)
}

func TestParseNoteDesc_Rejects(t *testing.T) {
	tests := []struct {
		name string
		desc []byte
	}{
		{name: "too short for the addresses", desc: make([]byte, 8)},
		{name: "empty", desc: nil},
		{
			name: "unterminated provider",
			desc: append(make([]byte, 3*8), []byte("compass")...), // no null terminator
		},
		{
			name: "unterminated probe",
			desc: func() []byte {
				d := make([]byte, 3*8)
				d = append(d, []byte("compass")...)
				d = append(d, 0)
				d = append(d, []byte("fpm_function")...) // no terminator
				return d
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseNoteDesc(tt.desc, 8, binary.LittleEndian)
			assert.Error(t, err)
		})
	}
}

// FuzzParseNoteDesc asserts no descriptor body, however malformed, can panic.
func FuzzParseNoteDesc(f *testing.F) {
	f.Add(makeDesc(0x1000, 0x2000, 0x3000, "compass", "fpm_function", "-8@%rax"))
	f.Add(make([]byte, 3*8))
	f.Add([]byte{})
	f.Add([]byte("compass\x00fpm_function\x00"))

	f.Fuzz(func(_ *testing.T, desc []byte) {
		// The contract under test: parseNoteDesc returns, it never panics.
		_, _ = parseNoteDesc(desc, 8, binary.LittleEndian)
	})
}
