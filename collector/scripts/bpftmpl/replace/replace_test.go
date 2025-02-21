package replace

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skpr/compass/collector/scripts/bpftmpl/elf"
)

func TestUsingNotesAmd64(t *testing.T) {
	notes := []elf.SystemTapNote{
		{
			Provider: "compass",
			Name:     "request_shutdown",
			Args: []string{
				"-8@%rbx",
				"-8@%r14",
				"-8@%rdi",
			},
		},
		{
			Provider: "compass",
			Name:     "php_function",
			Args: []string{
				"-8@%rbx",
				"-8@%r14",
				"-8@%rax",
				"-8@%rbp",
			},
		},
	}

	replacements := []string{
		"PHP_FUNCTION_ARG_FUNCTION_NAME",
		"PHP_FUNCTION_ARG_CLASS_NAME",
		"PHP_FUNCTION_ARG_FUNCTION_NAME",
		"PHP_FUNCTION_ARG_ELAPSED",
		"REQUEST_SHUTDOWN_ARG_REQUEST_ID",
		"REQUEST_SHUTDOWN_ARG_URI",
		"REQUEST_SHUTDOWN_ARG_URI",
	}

	program, err := UsingNotes("amd64", notes, strings.Join(replacements, ","))
	assert.NoError(t, err)
	assert.Equal(t, "ax,r14,ax,bp,bx,r14,r14", program)
}

func TestUsingNotesArm64(t *testing.T) {
	notes := []elf.SystemTapNote{
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
				"-8@x24",
			},
		},
	}

	replacements := []string{
		"PHP_FUNCTION_ARG_FUNCTION_NAME",
		"PHP_FUNCTION_ARG_CLASS_NAME",
		"PHP_FUNCTION_ARG_FUNCTION_NAME",
		"PHP_FUNCTION_ARG_ELAPSED",
		"REQUEST_SHUTDOWN_ARG_REQUEST_ID",
		"REQUEST_SHUTDOWN_ARG_URI",
		"REQUEST_SHUTDOWN_ARG_URI",
	}

	program, err := UsingNotes("arm64", notes, strings.Join(replacements, ","))
	assert.NoError(t, err)
	assert.Equal(t, "regs[0],regs[20],regs[0],regs[24],regs[19],regs[20],regs[20]", program)
}
