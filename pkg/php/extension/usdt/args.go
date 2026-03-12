package usdt

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Arg represents a parsed USDT probe argument's location within pt_regs.
type Arg struct {
	// Offset is the byte offset of the register within struct pt_regs.
	Offset uint32
	// Size is the size in bytes of the argument (negative means signed).
	Size int
}

// ParseArgs parses a USDT argument descriptor string from .note.stapsdt into a
// slice of Arg values. The descriptor format is a space-separated list of tokens
// like "-8@%rax -8@%rcx 8@%r15" (amd64) or "-8@x8 -8@x21 8@x0" (arm64). An
// empty string returns a nil slice without error.
func ParseArgs(args string) ([]Arg, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return nil, nil
	}

	tokens := strings.Fields(args)
	result := make([]Arg, 0, len(tokens))

	for _, token := range tokens {
		arg, err := parseArg(token)
		if err != nil {
			return nil, fmt.Errorf("parsing arg %q: %w", token, err)
		}
		result = append(result, arg)
	}

	return result, nil
}

// parseArg parses a single USDT argument descriptor token, e.g. "-8@%rax" or "8@x21".
func parseArg(token string) (Arg, error) {
	at := strings.IndexByte(token, '@')
	if at < 0 {
		return Arg{}, fmt.Errorf("missing '@' separator")
	}

	sizePart := token[:at]
	regPart := token[at+1:]

	size, err := strconv.Atoi(sizePart)
	if err != nil {
		return Arg{}, fmt.Errorf("invalid size %q: %w", sizePart, err)
	}
	if size == 0 {
		return Arg{}, fmt.Errorf("size must be non-zero")
	}

	// Strip leading '%' (x86 style) or nothing (arm64 style).
	regPart = strings.TrimPrefix(regPart, "%")

	offset, err := regOffset(regPart)
	if err != nil {
		return Arg{}, err
	}

	return Arg{Offset: offset, Size: size}, nil
}

// regOffset returns the byte offset of the named register within struct pt_regs.
// It handles both arm64 (xN / sp) and amd64 register names.
func regOffset(reg string) (uint32, error) {
	// arm64: registers are named x0..x30, plus sp.
	// regs[N] lives at offset N*8 inside struct pt_regs.
	if reg == "sp" {
		// sp is at regs[31], i.e. offset 248 on arm64; also valid for x86 (see table below).
		// Disambiguate: if it was already matched by the amd64 table that's fine — arm64
		// uses "sp" the same way, so the arm64 path wins here.
		return arm64RegisterOffset("sp")
	}
	if len(reg) >= 2 && reg[0] == 'x' && allDigits(reg[1:]) {
		return arm64RegisterOffset(reg)
	}

	// amd64: look up in the pt_regs struct field offset table.
	if off, ok := amd64RegOffsets[reg]; ok {
		return off, nil
	}

	return 0, fmt.Errorf("unknown register %q", reg)
}

// arm64RegisterOffset computes the byte offset for an arm64 register name.
// xN  → N * 8   (N = 0..30)
// sp  → 31 * 8 = 248
func arm64RegisterOffset(reg string) (uint32, error) {
	if reg == "sp" {
		return 31 * 8, nil
	}
	if len(reg) < 2 || reg[0] != 'x' {
		return 0, fmt.Errorf("unknown arm64 register %q", reg)
	}
	n, err := strconv.ParseUint(reg[1:], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid arm64 register number in %q: %w", reg, err)
	}
	if n > 30 {
		return 0, fmt.Errorf("arm64 register index %d out of range (0-30)", n)
	}
	return uint32(n * 8), nil
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

// amd64RegOffsets maps x86_64 register names (as they appear in stapsdt arg
// descriptors, after stripping the leading '%') to byte offsets within the
// Linux x86_64 struct pt_regs:
//
//	struct pt_regs {
//	    unsigned long r15;      // 0
//	    unsigned long r14;      // 8
//	    unsigned long r13;      // 16
//	    unsigned long r12;      // 24
//	    unsigned long rbp/bp;   // 32
//	    unsigned long rbx/bx;   // 40
//	    unsigned long r11;      // 48
//	    unsigned long r10;      // 56
//	    unsigned long r9;       // 64
//	    unsigned long r8;       // 72
//	    unsigned long rax/ax;   // 80
//	    unsigned long rcx/cx;   // 88
//	    unsigned long rdx/dx;   // 96
//	    unsigned long rsi/si;   // 104
//	    unsigned long rdi/di;   // 112
//	    unsigned long orig_rax; // 120
//	    unsigned long rip/ip;   // 128
//	    unsigned long cs;       // 136
//	    unsigned long eflags;   // 144
//	    unsigned long rsp/sp;   // 152
//	    unsigned long ss;       // 160
//	};
var amd64RegOffsets = map[string]uint32{
	// r15 family
	"r15": 0,
	// r14 family
	"r14": 8,
	// r13 family
	"r13": 16,
	// r12 family
	"r12": 24,
	// rbp family
	"rbp": 32, "ebp": 32, "bp": 32,
	// rbx family
	"rbx": 40, "ebx": 40, "bx": 40,
	// r11 family
	"r11": 48,
	// r10 family
	"r10": 56,
	// r9 family
	"r9": 64,
	// r8 family
	"r8": 72,
	// rax family
	"rax": 80, "eax": 80, "ax": 80, "al": 80,
	// rcx family
	"rcx": 88, "ecx": 88, "cx": 88, "cl": 88,
	// rdx family
	"rdx": 96, "edx": 96, "dx": 96, "dl": 96,
	// rsi family
	"rsi": 104, "esi": 104, "si": 104, "sil": 104,
	// rdi family
	"rdi": 112, "edi": 112, "di": 112, "dil": 112,
	// orig_rax
	"orig_rax": 120,
	// rip family
	"rip": 128, "eip": 128, "ip": 128,
	// cs
	"cs": 136,
	// eflags
	"eflags": 144,
	// rsp family
	"rsp": 152, "esp": 152, "sp": 152,
	// ss
	"ss": 160,
}
