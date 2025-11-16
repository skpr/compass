// Package main is used to generate a multi arch bpf program.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/skpr/compass/tracing/collector/scripts/bpftmpl/elf"
	"github.com/skpr/compass/tracing/collector/scripts/bpftmpl/replace"
)

func main() {
	var (
		flagArch     = flag.String("arch", "", "architecture which we will build with")
		flagTemplate = flag.String("template", "", "path to the bpf program template")
	)

	flag.Parse()

	extension, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	template, err := os.ReadFile(*flagTemplate)
	if err != nil {
		panic(err)
	}

	// Parse the output
	notes, err := elf.ReadNotes(string(extension))
	if err != nil {
		panic(err)
	}

	program, err := replace.UsingNotes(*flagArch, notes, string(template))
	if err != nil {
		panic(err)
	}

	// Write the file contents to stdout.
	fmt.Print(program)
}
