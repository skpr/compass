// Package pod resolves a Kubernetes pod UID to the processes that belong to it
// on the local node, without contacting the Kubernetes API.
//
// A DaemonSet collector shares the host PID namespace, so it can see every
// process on the node. Nothing on a process names its pod directly, but the
// pod UID and container ID are both written into the process's cgroup path by
// the kubelet. Scanning /proc/<pid>/cgroup for the requested UID therefore maps
// a pod UID to its PIDs using only node-local state.
package pod

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

// DefaultProcRoot is where the host /proc is expected to be mounted.
const DefaultProcRoot = "/proc"

// DefaultCgroupRoot is where the host unified (cgroup v2) hierarchy is expected
// to be mounted. It is only used to resolve a cgroup id for the BPF filter and
// is optional: resolution still returns matches when it is absent.
const DefaultCgroupRoot = "/sys/fs/cgroup"

// podUIDPattern matches the "pod<uid>" token the kubelet writes into a cgroup
// path. The UID is a UUID whose separators the systemd cgroup driver rewrites
// from dashes to underscores, so both are accepted and normalised afterwards.
var podUIDPattern = regexp.MustCompile(`pod([0-9a-fA-F]{8}[-_][0-9a-fA-F]{4}[-_][0-9a-fA-F]{4}[-_][0-9a-fA-F]{4}[-_][0-9a-fA-F]{12})`)

// Match is a single process found to belong to the target pod.
type Match struct {
	// PID of the process, as seen from the host PID namespace.
	PID int
	// Root is the path to the process's mount namespace root
	// (<procRoot>/<pid>/root), under which the instrumented binary is found.
	Root string
	// CgroupPath is the unified (cgroup v2) path from /proc/<pid>/cgroup, or
	// empty when the process only has v1 hierarchies.
	CgroupPath string
	// PIDNamespace is the inode of /proc/<pid>/ns/pid, used as a BPF filter key
	// on both cgroup v1 and v2. Zero when it could not be read.
	PIDNamespace uint64
	// CgroupID is the inode of the process's unified cgroup directory, used as a
	// BPF filter key on cgroup v2. Zero when it could not be resolved.
	CgroupID uint64
}

// Resolver maps a pod UID to its processes by scanning a /proc tree. The roots
// are injectable so the scan can be exercised against a fixture tree in tests.
type Resolver struct {
	// ProcRoot is the /proc mount to scan. Defaults to DefaultProcRoot.
	ProcRoot string
	// CgroupRoot is the unified cgroup mount used to resolve cgroup ids.
	// Defaults to DefaultCgroupRoot.
	CgroupRoot string
}

// NewResolver returns a Resolver using the default host mount points.
func NewResolver() *Resolver {
	return &Resolver{ProcRoot: DefaultProcRoot, CgroupRoot: DefaultCgroupRoot}
}

func (r *Resolver) procRoot() string {
	if r.ProcRoot == "" {
		return DefaultProcRoot
	}

	return r.ProcRoot
}

func (r *Resolver) cgroupRoot() string {
	if r.CgroupRoot == "" {
		return DefaultCgroupRoot
	}

	return r.CgroupRoot
}

// Resolve returns every process on the node whose cgroup path carries the given
// pod UID. An empty slice with a nil error means the pod has no processes here,
// which is the normal case for a node the pod is not scheduled on.
func (r *Resolver) Resolve(uid string) ([]Match, error) {
	want := NormaliseUID(uid)
	if want == "" {
		return nil, fmt.Errorf("invalid pod UID: %q", uid)
	}

	entries, err := os.ReadDir(r.procRoot())
	if err != nil {
		return nil, fmt.Errorf("failed to read proc root %s: %w", r.procRoot(), err)
	}

	var matches []Match

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			// Not a process directory (e.g. /proc/self, /proc/meminfo).
			continue
		}

		content, err := os.ReadFile(filepath.Join(r.procRoot(), entry.Name(), "cgroup"))
		if err != nil {
			// The process may have exited between the directory read and here,
			// or be one we cannot read; skip it rather than failing the scan.
			continue
		}

		got, ok := ExtractPodUID(string(content))
		if !ok || got != want {
			continue
		}

		matches = append(matches, r.describe(pid, string(content)))
	}

	return matches, nil
}

// describe gathers the filter keys and paths for a matched process. Missing
// keys are left zero: a scan that found the process is still useful even when
// one namespace or cgroup lookup fails.
func (r *Resolver) describe(pid int, cgroupContent string) Match {
	base := filepath.Join(r.procRoot(), strconv.Itoa(pid))

	m := Match{
		PID:          pid,
		Root:         filepath.Join(base, "root"),
		CgroupPath:   unifiedCgroupPath(cgroupContent),
		PIDNamespace: namespaceInode(filepath.Join(base, "ns", "pid")),
	}

	if m.CgroupPath != "" {
		m.CgroupID = inode(filepath.Join(r.cgroupRoot(), m.CgroupPath))
	}

	return m
}

// NormaliseUID lower-cases a pod UID and rewrites the systemd underscore
// separators back to the canonical dash form, so UIDs from either cgroup driver
// compare equal.
func NormaliseUID(uid string) string {
	uid = strings.TrimSpace(uid)
	if !podUIDPattern.MatchString("pod" + uid) {
		return ""
	}

	return strings.ToLower(strings.ReplaceAll(uid, "_", "-"))
}

// ExtractPodUID returns the normalised pod UID carried by a /proc/<pid>/cgroup
// file, if it has one. The file may list several hierarchies (cgroup v1) or a
// single unified line (cgroup v2); the first pod token found wins, as every
// hierarchy for a process agrees on its pod.
func ExtractPodUID(cgroupContent string) (string, bool) {
	m := podUIDPattern.FindStringSubmatch(cgroupContent)
	if m == nil {
		return "", false
	}

	return strings.ToLower(strings.ReplaceAll(m[1], "_", "-")), true
}

// unifiedCgroupPath returns the path from the cgroup v2 unified line ("0::"),
// or empty when the process only has v1 hierarchies.
func unifiedCgroupPath(cgroupContent string) string {
	for _, line := range strings.Split(cgroupContent, "\n") {
		// A cgroup line is hierarchy-id:controllers:path. The unified hierarchy
		// has an empty controller list and hierarchy id 0.
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}

		if parts[0] == "0" && parts[1] == "" {
			return parts[2]
		}
	}

	return ""
}

// namespaceInode reads the inode a namespace symlink points at, e.g.
// /proc/<pid>/ns/pid -> "pid:[4026531836]". Zero on any failure.
func namespaceInode(path string) uint64 {
	return inode(path)
}

// inode returns the inode number of a path, following symlinks. Zero on any
// failure, so a caller can treat "unknown" and "missing" alike.
func inode(path string) uint64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}

	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}

	return st.Ino
}
