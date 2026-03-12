// Package usdt provides the ability to attach probes to a binary.
package usdt

import (
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

// AttachProbe to the specified executable.
func AttachProbe(ex *link.Executable, executable, provider, probe string, prog *ebpf.Program) (link.Link, error) {
	locationFunctionBegin, err := getLocationFromProbe(executable, provider, probe)
	if err != nil {
		return nil, fmt.Errorf("failed to get probe location: %s: %w", probe, err)
	}

	return ex.Uprobe(getSymbol(provider, probe), prog, &link.UprobeOptions{
		Address:      locationFunctionBegin.Location,
		RefCtrOffset: locationFunctionBegin.SemaphoreOffsetRefctr,
	})
}

// GetProbeArgs parses the USDT argument descriptors from the .note.stapsdt
// section of the given ELF binary and returns a map of probe name → []Arg for
// each of the requested probes. The offsets in each Arg describe where to find
// that argument's value within struct pt_regs at probe fire time.
func GetProbeArgs(executable, provider string, probes []string) (map[string][]Arg, error) {
	result := make(map[string][]Arg, len(probes))

	for _, probe := range probes {
		note, err := getLocationFromProbe(executable, provider, probe)
		if err != nil {
			return nil, fmt.Errorf("failed to get probe location: %s: %w", probe, err)
		}

		args, err := ParseArgs(note.Args)
		if err != nil {
			return nil, fmt.Errorf("failed to parse args for probe %s: %w", probe, err)
		}

		result[probe] = args
	}

	return result, nil
}
