package pod

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormaliseUID(t *testing.T) {
	const canonical = "12345678-1234-1234-1234-123456789abc"

	cases := map[string]struct {
		in   string
		want string
	}{
		"dashed":        {"12345678-1234-1234-1234-123456789abc", canonical},
		"underscored":   {"12345678_1234_1234_1234_123456789abc", canonical},
		"uppercase":     {"12345678-1234-1234-1234-123456789ABC", canonical},
		"whitespace":    {"  12345678-1234-1234-1234-123456789abc\n", canonical},
		"empty":         {"", ""},
		"not-a-uuid":    {"not-a-uuid", ""},
		"too-short":     {"12345678-1234-1234-1234-123456789ab", ""},
		"path-injected": {"../etc", ""},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, NormaliseUID(tc.in))
		})
	}
}

func TestExtractPodUID(t *testing.T) {
	const canonical = "12345678-1234-1234-1234-123456789abc"

	cases := map[string]struct {
		content string
		want    string
		ok      bool
	}{
		"cgroupfs v2 burstable": {
			content: "0::/kubepods/burstable/pod12345678-1234-1234-1234-123456789abc/abc123def456\n",
			want:    canonical,
			ok:      true,
		},
		"systemd v2 besteffort": {
			content: "0::/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod12345678_1234_1234_1234_123456789abc.slice/cri-containerd-abc123.scope\n",
			want:    canonical,
			ok:      true,
		},
		"systemd v2 guaranteed (no qos segment)": {
			content: "0::/kubepods.slice/kubepods-pod12345678_1234_1234_1234_123456789abc.slice/cri-containerd-abc123.scope\n",
			want:    canonical,
			ok:      true,
		},
		"cgroup v1 multiline": {
			content: "12:pids:/kubepods/burstable/pod12345678-1234-1234-1234-123456789abc/abc\n" +
				"11:memory:/kubepods/burstable/pod12345678-1234-1234-1234-123456789abc/abc\n",
			want: canonical,
			ok:   true,
		},
		"no pod token": {
			content: "0::/system.slice/sshd.service\n",
			ok:      false,
		},
		"empty": {
			content: "",
			ok:      false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok := ExtractPodUID(tc.content)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestUnifiedCgroupPath(t *testing.T) {
	v2 := "0::/kubepods.slice/kubepods-pod123.slice/cri-containerd-abc.scope\n"
	assert.Equal(t, "/kubepods.slice/kubepods-pod123.slice/cri-containerd-abc.scope", unifiedCgroupPath(v2))

	v1 := "12:pids:/kubepods/burstable/podX/abc\n11:memory:/kubepods/burstable/podX/abc\n"
	assert.Equal(t, "", unifiedCgroupPath(v1))
}

// writeProc creates <procRoot>/<pid>/cgroup with the given content and a
// ns/pid file standing in for the namespace symlink.
func writeProc(t *testing.T, procRoot string, pid int, cgroup string) {
	t.Helper()

	dir := filepath.Join(procRoot, strconv.Itoa(pid))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "ns"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "root"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cgroup"), []byte(cgroup), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ns", "pid"), []byte("ns"), 0644))
}

func TestResolve(t *testing.T) {
	const uid = "12345678-1234-1234-1234-123456789abc"

	procRoot := t.TempDir()
	cgroupRoot := t.TempDir()

	// Two processes in the target pod (e.g. an fpm master and a worker).
	target := "0::/kubepods.slice/kubepods-pod12345678_1234_1234_1234_123456789abc.slice/cri-containerd-runtime.scope\n"
	writeProc(t, procRoot, 100, target)
	writeProc(t, procRoot, 101, target)

	// A process in a different pod.
	writeProc(t, procRoot, 200, "0::/kubepods.slice/kubepods-podffffffff_ffff_ffff_ffff_ffffffffffff.slice/cri-containerd-other.scope\n")

	// A system process with no pod token.
	writeProc(t, procRoot, 300, "0::/system.slice/sshd.service\n")

	// Non-process entries that must be skipped.
	require.NoError(t, os.MkdirAll(filepath.Join(procRoot, "self"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(procRoot, "meminfo"), []byte("x"), 0644))

	// Provide the unified cgroup directory so a cgroup id resolves.
	require.NoError(t, os.MkdirAll(filepath.Join(cgroupRoot, "kubepods.slice/kubepods-pod12345678_1234_1234_1234_123456789abc.slice/cri-containerd-runtime.scope"), 0755))

	r := &Resolver{ProcRoot: procRoot, CgroupRoot: cgroupRoot}

	matches, err := r.Resolve(uid)
	require.NoError(t, err)

	pids := make([]int, 0, len(matches))
	for _, m := range matches {
		pids = append(pids, m.PID)
	}
	sort.Ints(pids)

	assert.Equal(t, []int{100, 101}, pids, "only the target pod's processes should match")

	for _, m := range matches {
		assert.Equal(t, filepath.Join(procRoot, strconv.Itoa(m.PID), "root"), m.Root)
		assert.NotZero(t, m.PIDNamespace, "namespace inode should be populated")
		assert.NotZero(t, m.CgroupID, "cgroup id should resolve from the provided cgroup root")
	}
}

func TestResolve_AcceptsUnderscoreUID(t *testing.T) {
	procRoot := t.TempDir()

	writeProc(t, procRoot, 100, "0::/kubepods.slice/kubepods-pod12345678_1234_1234_1234_123456789abc.slice/cri-containerd-x.scope\n")

	r := &Resolver{ProcRoot: procRoot}

	// Query with the canonical dashed form; the cgroup uses underscores.
	matches, err := r.Resolve("12345678-1234-1234-1234-123456789abc")
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, 100, matches[0].PID)
}

func TestResolve_NoMatchesIsNotAnError(t *testing.T) {
	procRoot := t.TempDir()
	writeProc(t, procRoot, 100, "0::/system.slice/sshd.service\n")

	r := &Resolver{ProcRoot: procRoot}

	matches, err := r.Resolve("12345678-1234-1234-1234-123456789abc")
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestResolve_RejectsInvalidUID(t *testing.T) {
	r := &Resolver{ProcRoot: t.TempDir()}

	_, err := r.Resolve("not-a-uuid")
	assert.Error(t, err)
}

// TestResolve_CachesNegativeResult proves a no-match result is remembered for
// the negative TTL: a pod added to the tree after a miss is not seen until the
// cached miss expires. This is what spares the daemon a full /proc rescan on
// every retry for a pod that is not (yet) on the node.
func TestResolve_CachesNegativeResult(t *testing.T) {
	const uid = "12345678-1234-1234-1234-123456789abc"
	target := "0::/kubepods.slice/kubepods-pod12345678_1234_1234_1234_123456789abc.slice/cri-containerd-x.scope\n"

	procRoot := t.TempDir()

	now := time.Unix(0, 0)
	r := &Resolver{ProcRoot: procRoot, NegativeTTL: 5 * time.Second, now: func() time.Time { return now }}

	// Absent pod: miss, cached.
	matches, err := r.Resolve(uid)
	require.NoError(t, err)
	require.Empty(t, matches)

	// The pod appears, but within the TTL the cached miss is still returned.
	writeProc(t, procRoot, 100, target)

	matches, err = r.Resolve(uid)
	require.NoError(t, err)
	assert.Empty(t, matches, "a cached miss should be reused within the negative TTL")

	// After the TTL, the scan runs again and finds the pod.
	now = now.Add(6 * time.Second)

	matches, err = r.Resolve(uid)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, 100, matches[0].PID)
}

// TestResolve_DoesNotCacheHits confirms a successful resolution is never cached,
// so a caller always gets a current process list (fresh PIDs).
func TestResolve_DoesNotCacheHits(t *testing.T) {
	const uid = "12345678-1234-1234-1234-123456789abc"
	target := "0::/kubepods.slice/kubepods-pod12345678_1234_1234_1234_123456789abc.slice/cri-containerd-x.scope\n"

	procRoot := t.TempDir()
	writeProc(t, procRoot, 100, target)

	now := time.Unix(0, 0)
	r := &Resolver{ProcRoot: procRoot, NegativeTTL: time.Hour, now: func() time.Time { return now }}

	matches, err := r.Resolve(uid)
	require.NoError(t, err)
	require.Len(t, matches, 1)

	// The worker recycles to a new PID; the next resolve must reflect it despite
	// the long TTL, because hits are not cached.
	require.NoError(t, os.RemoveAll(filepath.Join(procRoot, "100")))
	writeProc(t, procRoot, 101, target)

	matches, err = r.Resolve(uid)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, 101, matches[0].PID, "a successful resolution must not be served from cache")
}

// TestResolve_NegativeTTLDisabled confirms a negative TTL turns the cache off.
func TestResolve_NegativeTTLDisabled(t *testing.T) {
	const uid = "12345678-1234-1234-1234-123456789abc"
	target := "0::/kubepods.slice/kubepods-pod12345678_1234_1234_1234_123456789abc.slice/cri-containerd-x.scope\n"

	procRoot := t.TempDir()
	r := &Resolver{ProcRoot: procRoot, NegativeTTL: -1}

	matches, err := r.Resolve(uid)
	require.NoError(t, err)
	require.Empty(t, matches)

	writeProc(t, procRoot, 100, target)

	// With caching disabled, the pod is found immediately on the next call.
	matches, err = r.Resolve(uid)
	require.NoError(t, err)
	require.Len(t, matches, 1)
}

// TestInode sanity-checks that inode() returns the same number the kernel
// reports for a real file, so the filter keys are meaningful.
func TestInode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))

	info, err := os.Stat(path)
	require.NoError(t, err)
	want := info.Sys().(*syscall.Stat_t).Ino

	assert.Equal(t, want, inode(path))
	assert.Zero(t, inode(filepath.Join(t.TempDir(), "missing")))
}
