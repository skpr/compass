// Package cgroupfilter restricts eBPF trace collection to a set of cgroup ids.
//
// The sidecar traces its whole pod, so it uses an empty filter and every probe
// fires. The DaemonSet collector shares the host with many pods and attaches to
// a binary whose inode overlayfs may share across pods, so it must drop events
// from cgroups other than the target pod's. The BPF programs carry a
// filter_by_cgroup switch and an allowed_cgroups map for exactly this; this
// package is the user-space half that sets them.
package cgroupfilter

import (
	"fmt"

	"github.com/cilium/ebpf"
)

const (
	// VarName is the BPF constant that switches cgroup filtering on. It is left
	// zero (off) by default so a program that does not populate the map keeps
	// tracing everything.
	VarName = "filter_by_cgroup"

	// MapName is the BPF hash map of allowed cgroup ids.
	MapName = "allowed_cgroups"
)

// Filter is the set of cgroup ids permitted to emit events. An empty filter
// disables filtering entirely, which is the sidecar's behaviour.
type Filter struct {
	// AllowedCgroupIDs are the cgroup v2 ids (directory inode numbers) whose
	// tasks may emit events.
	AllowedCgroupIDs []uint64
}

// Enabled reports whether any filtering will be applied.
func (f Filter) Enabled() bool {
	return len(f.AllowedCgroupIDs) > 0
}

// Apply writes the filter switch into the collection spec before load. It must
// be called before the spec is loaded into the kernel, as the switch is a
// compile-time-style constant baked in at load.
func (f Filter) Apply(spec *ebpf.CollectionSpec) error {
	v, ok := spec.Variables[VarName]
	if !ok {
		// A program built before the filter existed simply cannot be filtered;
		// that is only ever the sidecar, which never enables it.
		if f.Enabled() {
			return fmt.Errorf("cgroup filtering requested but BPF variable %q is absent", VarName)
		}

		return nil
	}

	on := uint8(0)
	if f.Enabled() {
		on = 1
	}

	if err := v.Set(on); err != nil {
		return fmt.Errorf("failed to set %q: %w", VarName, err)
	}

	return nil
}

// Populate loads the allowed cgroup ids into the map after the program is
// loaded. It is a no-op when the filter is disabled.
func (f Filter) Populate(m *ebpf.Map) error {
	if !f.Enabled() {
		return nil
	}

	if m == nil {
		return fmt.Errorf("cgroup filtering requested but the %q map is nil", MapName)
	}

	for _, id := range f.AllowedCgroupIDs {
		if err := m.Put(id, uint8(1)); err != nil {
			return fmt.Errorf("failed to add cgroup id %d to %q: %w", id, MapName, err)
		}
	}

	return nil
}
