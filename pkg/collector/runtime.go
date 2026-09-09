package collector

import (
	"os"
	"path/filepath"

	"github.com/skpr/compass/pkg/collector/pod"
)

// Runtimes describes the instrumented runtimes found inside a target pod and
// the cgroup ids that scope the eBPF filter to that pod.
type Runtimes struct {
	// PHPExtensionPath is the collector-visible path to the PHP extension inside
	// the target container (rewritten through the process root), or empty when
	// the pod has no PHP runtime.
	PHPExtensionPath string
	// NodeAddonPath is the collector-visible path to the Node addon inside the
	// target container, or empty when the pod has no Node runtime.
	NodeAddonPath string
	// AllowedCgroupIDs are the distinct cgroup ids of the pod's processes, used
	// to restrict the eBPF probes to this pod.
	AllowedCgroupIDs []uint64
}

// Empty reports whether no instrumented runtime was found in the pod.
func (r Runtimes) Empty() bool {
	return r.PHPExtensionPath == "" && r.NodeAddonPath == ""
}

// LocateRuntimes inspects a pod's resolved processes and returns the paths to
// whichever instrumented runtimes it carries, together with the cgroup ids that
// identify the pod for the eBPF filter.
//
// phpExtensionPath and nodeAddonPath are the in-container paths, as the
// application sees them. Each is rewritten through a process's root
// (/proc/<pid>/root/...) so the collector, which runs in a different mount
// namespace, can open the file.
func LocateRuntimes(matches []pod.Match, phpExtensionPath, nodeAddonPath string) Runtimes {
	var r Runtimes

	seen := make(map[uint64]struct{})

	for _, m := range matches {
		if m.CgroupID != 0 {
			if _, ok := seen[m.CgroupID]; !ok {
				seen[m.CgroupID] = struct{}{}
				r.AllowedCgroupIDs = append(r.AllowedCgroupIDs, m.CgroupID)
			}
		}

		if r.PHPExtensionPath == "" && phpExtensionPath != "" {
			if candidate := filepath.Join(m.Root, phpExtensionPath); regularFile(candidate) {
				r.PHPExtensionPath = candidate
			}
		}

		if r.NodeAddonPath == "" && nodeAddonPath != "" {
			if candidate := filepath.Join(m.Root, nodeAddonPath); regularFile(candidate) {
				r.NodeAddonPath = candidate
			}
		}
	}

	return r
}

func regularFile(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.Mode().IsRegular()
}
