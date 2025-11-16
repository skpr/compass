package elf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadNotes(t *testing.T) {
	input := `Displaying notes found in: .note.gnu.build-id
  Owner                Data size        Description
  GNU                  0x00000014       NT_GNU_BUILD_ID (unique build ID bitstring)
    Build ID: b7cd3dc63ad11bacdfbc7abe8d404b83d62b0b5c

Displaying notes found in: .note.stapsdt
  Owner                Data size        Description
  stapsdt              0x00000044       NT_STAPSDT (SystemTap probe descriptors)
    Provider: compass
    Name: request_shutdown
    Location: 0x000000000000c9f8, Base: 0x0000000000058393, Semaphore: 0x00000000000801c6
    Arguments: -8@x8 -8@x21 -8@x0
  stapsdt              0x00000041       NT_STAPSDT (SystemTap probe descriptors)
    Provider: compass
    Name: php_function
    Location: 0x000000000000d8e8, Base: 0x0000000000058393, Semaphore: 0x00000000000801c8
    Arguments: -8@x20 -8@x0 -8@x21`

	have, err := ReadNotes(input)
	assert.NoError(t, err)

	want := []SystemTapNote{
		{
			Provider: "compass",
			Name:     "request_shutdown",
			Args: []string{
				"-8@x8",
				"-8@x21",
				"-8@x0",
			},
		},
		{
			Provider: "compass",
			Name:     "php_function",
			Args: []string{
				"-8@x20",
				"-8@x0",
				"-8@x21",
			},
		},
	}

	assert.Equal(t, want, have)
}
