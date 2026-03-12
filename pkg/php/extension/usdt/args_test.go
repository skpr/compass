package usdt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArgs_amd64(t *testing.T) {
	// Typical args string for a probe compiled on x86_64.
	// Format: <size>@<register> with leading '%'.
	args, err := ParseArgs("-8@%rax -8@%rcx 8@%r15")
	require.NoError(t, err)
	require.Len(t, args, 3)

	assert.Equal(t, uint32(80), args[0].Offset) // rax
	assert.Equal(t, -8, args[0].Size)

	assert.Equal(t, uint32(88), args[1].Offset) // rcx
	assert.Equal(t, -8, args[1].Size)

	assert.Equal(t, uint32(0), args[2].Offset) // r15
	assert.Equal(t, 8, args[2].Size)
}

func TestParseArgs_amd64_aliases(t *testing.T) {
	// Compilers may emit short-form register names.
	args, err := ParseArgs("-8@%ax -8@%cx 8@%r14")
	require.NoError(t, err)
	require.Len(t, args, 3)

	assert.Equal(t, uint32(80), args[0].Offset) // ax  → rax
	assert.Equal(t, uint32(88), args[1].Offset) // cx  → rcx
	assert.Equal(t, uint32(8), args[2].Offset)  // r14
}

func TestParseArgs_arm64(t *testing.T) {
	// Typical args string for a probe compiled on arm64.
	// Format: <size>@<xN> without a '%' prefix.
	args, err := ParseArgs("-8@x8 -8@x21 8@x0")
	require.NoError(t, err)
	require.Len(t, args, 3)

	assert.Equal(t, uint32(64), args[0].Offset) // x8  → 8*8
	assert.Equal(t, -8, args[0].Size)

	assert.Equal(t, uint32(168), args[1].Offset) // x21 → 21*8
	assert.Equal(t, -8, args[1].Size)

	assert.Equal(t, uint32(0), args[2].Offset) // x0  → 0*8
	assert.Equal(t, 8, args[2].Size)
}

func TestParseArgs_arm64_single(t *testing.T) {
	args, err := ParseArgs("-8@x8")
	require.NoError(t, err)
	require.Len(t, args, 1)
	assert.Equal(t, uint32(64), args[0].Offset)
	assert.Equal(t, -8, args[0].Size)
}

func TestParseArgs_empty(t *testing.T) {
	args, err := ParseArgs("")
	require.NoError(t, err)
	assert.Nil(t, args)
}

func TestParseArgs_whitespaceOnly(t *testing.T) {
	args, err := ParseArgs("   ")
	require.NoError(t, err)
	assert.Nil(t, args)
}

func TestParseArgs_unknownRegister(t *testing.T) {
	_, err := ParseArgs("-8@%zz99")
	assert.ErrorContains(t, err, "unknown register")
}

func TestParseArgs_missingAt(t *testing.T) {
	_, err := ParseArgs("-8rax")
	assert.ErrorContains(t, err, "missing '@'")
}

func TestParseArgs_invalidSize(t *testing.T) {
	_, err := ParseArgs("xyz@%rax")
	assert.ErrorContains(t, err, "invalid size")
}

func TestParseArgs_zeroSize(t *testing.T) {
	_, err := ParseArgs("0@%rax")
	assert.ErrorContains(t, err, "non-zero")
}

func TestRegOffset_arm64Boundary(t *testing.T) {
	// x0 → 0, x30 → 240
	off, err := regOffset("x0")
	require.NoError(t, err)
	assert.Equal(t, uint32(0), off)

	off, err = regOffset("x30")
	require.NoError(t, err)
	assert.Equal(t, uint32(240), off)
}

func TestRegOffset_arm64OutOfRange(t *testing.T) {
	_, err := regOffset("x31")
	assert.ErrorContains(t, err, "out of range")
}

func TestRegOffset_amd64All(t *testing.T) {
	tests := []struct {
		reg    string
		offset uint32
	}{
		{"r15", 0}, {"r14", 8}, {"r13", 16}, {"r12", 24},
		{"rbp", 32}, {"rbx", 40}, {"r11", 48}, {"r10", 56},
		{"r9", 64}, {"r8", 72},
		{"rax", 80}, {"eax", 80}, {"ax", 80},
		{"rcx", 88}, {"ecx", 88}, {"cx", 88},
		{"rdx", 96}, {"rsi", 104}, {"rdi", 112},
		{"rip", 128}, {"rsp", 152},
	}
	for _, tt := range tests {
		off, err := regOffset(tt.reg)
		require.NoError(t, err, "reg=%s", tt.reg)
		assert.Equal(t, tt.offset, off, "reg=%s", tt.reg)
	}
}
