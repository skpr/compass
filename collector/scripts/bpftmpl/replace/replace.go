package replace

import (
	"fmt"
	"strings"

	"github.com/skpr/compass/collector/scripts/bpftmpl/elf"
)

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
			if len(note.Args) != 5 {
				return "", fmt.Errorf("php_fuction does not have 5 args")
			}

			replacements["PHP_FUNCTION_ARG_REQUEST_ID"] = valueFunc(note.Args[0])
			replacements["PHP_FUNCTION_ARG_CLASS_NAME"] = valueFunc(note.Args[1])
			replacements["PHP_FUNCTION_ARG_FUNCTION_NAME"] = valueFunc(note.Args[2])
			replacements["PHP_FUNCTION_ARG_START_TIME"] = valueFunc(note.Args[3])
			replacements["PHP_FUNCTION_ARG_END_TIME"] = valueFunc(note.Args[4])

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
			// @todo, Determine if there are any other prefixes other than "-8@x"
			return fmt.Sprintf("regs[%s]", strings.TrimLeft(argument, "-8@x"))
		}, nil
	}

	if arch == "amd64" {
		return func(argument string) string {
			// @todo, Determine if there are any other prefixes other than "-8@%"
			return strings.TrimLeft(argument, "-8@%")
		}, nil
	}

	return nil, fmt.Errorf("architecture not supported: %s", arch)
}
