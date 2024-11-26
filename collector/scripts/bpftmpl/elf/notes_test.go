package elf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadNotes(t *testing.T) {
	input := `Displaying notes found in: .note.gnu.build-id
  Owner                Data size        Description
  GNU                  0x00000014       NT_GNU_BUILD_ID (unique build ID bitstring)
    Build ID: 30d4ef247fb97a94101adda2291644ccdcb8cc77

Displaying notes found in: .note.stapsdt
  Owner                Data size        Description
  stapsdt              0x00000045       NT_STAPSDT (SystemTap probe descriptors)
    Provider: compass
    Name: request_shutdown
    Location: 0x000000000000cfa8, Base: 0x0000000000056537, Semaphore: 0x0000000000000000
    Arguments: -8@x19 -8@x20 -8@x0
  stapsdt              0x0000004f       NT_STAPSDT (SystemTap probe descriptors)
    Provider: compass
    Name: php_function
    Location: 0x000000000000e794, Base: 0x0000000000056537, Semaphore: 0x0000000000000000
    Arguments: -8@x19 -8@x20 -8@x0 -8@x23 -8@x24`

	have, err := ReadNotes(input)
	assert.NoError(t, err)

	want := []SystemTapNote{
		{
			Provider: "compass",
			Name:     "request_shutdown",
			Args: []string{
				"-8@x19",
				"-8@x20",
				"-8@x0",
			},
		},
		{
			Provider: "compass",
			Name:     "php_function",
			Args: []string{
				"-8@x19",
				"-8@x20",
				"-8@x0",
				"-8@x23",
				"-8@x24",
			},
		},
	}

	assert.Equal(t, want, have)
}
