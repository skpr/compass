package replace

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skpr/compass/tracing/collector/scripts/bpftmpl/elf"
)

func TestUsingNotesAmd64(t *testing.T) {
	notes := []elf.SystemTapNote{
		{
			Provider: "compass",
			Name:     "request_init",
			Args: []string{
				"-8@%rbx",
				"-8@%r14",
				"-8@%rdi",
			},
		},
		{
			Provider: "compass",
			Name:     "request_shutdown",
			Args: []string{
				"-8@%rbx",
			},
		},
		{
			Provider: "compass",
			Name:     "php_function",
			Args: []string{
				"-8@%rbx",
				"-8@%rax",
				"-8@%rbp",
			},
		},
	}

	replacements := []string{
		"REQUEST_INIT_ARG_REQUEST_ID",
		"REQUEST_INIT_ARG_URI",
		"REQUEST_INIT_ARG_METHOD",
		"PHP_FUNCTION_ARG_FUNCTION_NAME",
		"PHP_FUNCTION_ARG_ELAPSED",
		"REQUEST_SHUTDOWN_ARG_REQUEST_ID",
	}

	program, err := UsingNotes("amd64", notes, strings.Join(replacements, ","))
	assert.NoError(t, err)
	assert.Equal(t, "bx,r14,di,ax,bp,bx", program)
}

func TestUsingNotesArm64(t *testing.T) {
	notes := []elf.SystemTapNote{
		{
			Provider: "compass",
			Name:     "request_init",
			Args: []string{
				"-8@x19",
				"-8@x20",
				"-8@x0",
			},
		},
		{
			Provider: "compass",
			Name:     "request_shutdown",
			Args: []string{
				"-8@x19",
			},
		},
		{
			Provider: "compass",
			Name:     "php_function",
			Args: []string{
				"-8@x20",
				"-8@x0",
				"-8@x24",
			},
		},
	}

	replacements := []string{
		"REQUEST_INIT_ARG_REQUEST_ID",
		"REQUEST_INIT_ARG_URI",
		"REQUEST_INIT_ARG_METHOD",
		"PHP_FUNCTION_ARG_FUNCTION_NAME",
		"PHP_FUNCTION_ARG_ELAPSED",
		"REQUEST_SHUTDOWN_ARG_REQUEST_ID",
	}

	program, err := UsingNotes("arm64", notes, strings.Join(replacements, ","))
	assert.NoError(t, err)
	assert.Equal(t, "regs[19],regs[20],regs[0],regs[0],regs[24],regs[19]", program)
}
