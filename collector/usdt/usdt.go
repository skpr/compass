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
