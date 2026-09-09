package collector

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/collector/pod"
)

// makeRoot creates a fake container root containing the given relative files.
func makeRoot(t *testing.T, files ...string) string {
	t.Helper()

	root := t.TempDir()
	for _, f := range files {
		full := filepath.Join(root, f)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0755))
		require.NoError(t, os.WriteFile(full, []byte("x"), 0644))
	}

	return root
}

func TestLocateRuntimes_FindsPHP(t *testing.T) {
	const phpRel = "/usr/lib/php/modules/compass.so"

	root := makeRoot(t, phpRel)

	// Two PIDs in the same container share a cgroup id; a third is a sibling
	// container in the same pod with a different one.
	matches := []pod.Match{
		{PID: 100, Root: root, CgroupID: 111},
		{PID: 101, Root: root, CgroupID: 111},
		{PID: 102, Root: t.TempDir(), CgroupID: 222},
	}

	r := LocateRuntimes(matches, phpRel, "/usr/lib/compass/node/compass.node")

	assert.Equal(t, filepath.Join(root, phpRel), r.PHPExtensionPath)
	assert.Empty(t, r.NodeAddonPath)
	assert.False(t, r.Empty())

	ids := append([]uint64(nil), r.AllowedCgroupIDs...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	assert.Equal(t, []uint64{111, 222}, ids, "cgroup ids should be distinct across the pod")
}

func TestLocateRuntimes_FindsNode(t *testing.T) {
	const nodeRel = "/usr/lib/compass/node/compass.node"

	root := makeRoot(t, nodeRel)

	matches := []pod.Match{{PID: 100, Root: root, CgroupID: 5}}

	r := LocateRuntimes(matches, "/usr/lib/php/modules/compass.so", nodeRel)

	assert.Empty(t, r.PHPExtensionPath)
	assert.Equal(t, filepath.Join(root, nodeRel), r.NodeAddonPath)
}

func TestLocateRuntimes_NoRuntimeIsEmpty(t *testing.T) {
	matches := []pod.Match{{PID: 100, Root: t.TempDir(), CgroupID: 5}}

	r := LocateRuntimes(matches, "/usr/lib/php/modules/compass.so", "/usr/lib/compass/node/compass.node")

	assert.True(t, r.Empty())
	assert.Equal(t, []uint64{5}, r.AllowedCgroupIDs, "cgroup ids are collected even without a runtime")
}

func TestLocateRuntimes_IgnoresZeroCgroupID(t *testing.T) {
	root := makeRoot(t, "/x")
	matches := []pod.Match{{PID: 100, Root: root, CgroupID: 0}}

	r := LocateRuntimes(matches, "", "")

	assert.Empty(t, r.AllowedCgroupIDs, "a zero (unresolved) cgroup id must not be added to the allow list")
}
