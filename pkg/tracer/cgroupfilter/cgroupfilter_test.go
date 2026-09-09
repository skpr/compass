package cgroupfilter

import (
	"testing"

	"github.com/cilium/ebpf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterEnabled(t *testing.T) {
	assert.False(t, Filter{}.Enabled())
	assert.False(t, Filter{AllowedCgroupIDs: []uint64{}}.Enabled())
	assert.True(t, Filter{AllowedCgroupIDs: []uint64{42}}.Enabled())
}

func TestApply_MissingVariable(t *testing.T) {
	spec := &ebpf.CollectionSpec{Variables: map[string]*ebpf.VariableSpec{}}

	// A disabled filter tolerates a program without the switch (the sidecar's
	// older program), an enabled one does not.
	require.NoError(t, Filter{}.Apply(spec))

	err := Filter{AllowedCgroupIDs: []uint64{1}}.Apply(spec)
	assert.Error(t, err)
}

func TestPopulate_DisabledIsNoOp(t *testing.T) {
	// A disabled filter must not require a map, so an empty filter never touches
	// the kernel.
	assert.NoError(t, Filter{}.Populate(nil))
}

func TestPopulate_EnabledRequiresMap(t *testing.T) {
	err := Filter{AllowedCgroupIDs: []uint64{1}}.Populate(nil)
	assert.Error(t, err)
}
