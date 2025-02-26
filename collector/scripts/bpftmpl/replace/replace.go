// Package replace is used to find and replace program values.
package replace

import (
	"fmt"
	"strings"

	"github.com/skpr/compass/collector/scripts/bpftmpl/elf"
)

// UsingNotes replaces strings in the program based on provided notes.
func UsingNotes(arch string, notes []elf.SystemTapNote, program string) (string, error) {
	replacements := make(map[string]string, 8)

	valueFunc, err := getValueFunc(arch)
	if err != nil {
		return "", err
	}

	for _, note := range notes {
		if note.Provider != "compass" {
			return "", fmt.Errorf("found a note which is not provided by compass")
		}

		switch note.Name {
		case "php_function":
			if len(note.Args) != 3 {
				return "", fmt.Errorf("php_fuction does not have 3 args")
			}

			replacements["PHP_FUNCTION_ARG_REQUEST_ID"] = valueFunc(note.Args[0])
			replacements["PHP_FUNCTION_ARG_FUNCTION_NAME"] = valueFunc(note.Args[1])
			replacements["PHP_FUNCTION_ARG_ELAPSED"] = valueFunc(note.Args[2])

		case "request_shutdown":
			if len(note.Args) != 3 {
				return "", fmt.Errorf("request_shutdown does not have 3 args")
			}

			replacements["REQUEST_SHUTDOWN_ARG_REQUEST_ID"] = valueFunc(note.Args[0])
			replacements["REQUEST_SHUTDOWN_ARG_URI"] = valueFunc(note.Args[1])
			replacements["REQUEST_SHUTDOWN_ARG_METHOD"] = valueFunc(note.Args[2])

		default:
			return "", fmt.Errorf("found a note which is not php_function or request_shutdown")
		}
	}

	for key, value := range replacements {
		program = strings.ReplaceAll(program, key, value)
	}

	return program, nil
}

func getValueFunc(arch string) (func(string) string, error) {
	if arch == "arm64" {
		return func(argument string) string {
			return fmt.Sprintf("regs[%s]", strings.TrimPrefix(argument, "-8@x"))
		}, nil
	}

	if arch == "amd64" {
		return func(argument string) string {
			switch argument {
			case "-8@%rax":
				return "ax"
			case "-8@%rdi":
				return "di"
			case "-8@%rsi":
				return "ax"
			case "-8@%rdx":
				return "dx"
			case "-8@%rbx":
				return "bx"
			case "-8@%rbp":
				return "bp"
			case "-8@%rcx":
				return "cx"
			default:
				// Preserve the "r" in the remaining eg. r15.
				return strings.TrimPrefix(argument, "-8@%")
			}
		}, nil
	}

	return nil, fmt.Errorf("architecture not supported: %s", arch)
}
