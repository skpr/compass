package usdt

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrProbeNotFound is returned when the binary carries no note for the probe
// that was asked for. Probes which an older build of the instrumented binary
// may not have are attached on a best effort basis, so callers need to tell
// "this probe is absent" apart from "this binary could not be read".
var ErrProbeNotFound = errors.New("probe not found")

// maxNoteDescSize bounds a single stapsdt note descriptor. A real descriptor is
// three addresses plus a few short strings, so this is far larger than any
// legitimate note; it exists to reject a corrupt descsz before it is used to
// size an allocation.
const maxNoteDescSize int32 = 64 * 1024

// Note represents a SystemTap note.
type Note struct {
	Location              uint64
	Base                  uint64
	Semaphore             uint64
	SemaphoreOffsetPtrace uint64
	SemaphoreOffsetRefctr uint64
	// Args is the raw argument descriptor string from the .note.stapsdt section,
	// e.g. "-8@%rax -8@%rcx 8@%r15" on amd64 or "-8@x8 -8@x21 8@x0" on arm64.
	Args string
	bo   binary.ByteOrder
}

func getSymbol(provider, function string) string {
	return fmt.Sprintf("usdt_%s_%s", provider, function)
}

func getLocationFromProbe(path, provider, probe string) (*Note, error) {
	osf, err := os.Open(path)
	if err != nil {
		// Not an executable or shared object.
		return nil, err
	}

	f, err := elf.NewFile(osf)
	if err != nil {
		// Not an executable or shared object.
		return nil, err
	}
	defer f.Close()

	sec := f.Section(".note.stapsdt")
	if sec == nil {
		return nil, errors.New("SDT note section not found")
	}

	// Only 64-bit binaries are supported: the extension and addon are always
	// built 64-bit, and the address reads below assume eight-byte fields.
	if f.Class != elf.ELFCLASS64 {
		return nil, fmt.Errorf("unsupported ELF class %s: only 64-bit binaries are supported", f.Class)
	}

	addrsz := 8

	r := sec.Open()
	base := sdtBaseAddr(f)
	for {
		var namesz, descsz int32

		err = binary.Read(r, f.ByteOrder, &namesz)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if namesz < 0 {
			return nil, fmt.Errorf("malformed stapsdt note: negative namesz: %d", namesz)
		}

		err = binary.Read(r, f.ByteOrder, &descsz)
		if err != nil {
			return nil, err
		}

		// A valid descriptor holds at least the three addresses; reject anything
		// smaller or implausibly large before it sizes the allocation below, so a
		// corrupt note cannot wrap to a huge make or leave the field reads that
		// follow out of bounds.
		if descsz < int32(3*addrsz) || descsz > maxNoteDescSize {
			return nil, fmt.Errorf("malformed stapsdt note: descsz out of range: %d", descsz)
		}

		// skip note type
		_, err := r.Seek(4, io.SeekCurrent)
		if err != nil {
			return nil, err
		}

		// skip note name
		_, err = r.Seek(int64(namesz), io.SeekCurrent)
		if err != nil {
			return nil, err
		}

		align4 := func(n int32) uint64 {
			return (uint64(n) + 4 - 1) / 4 * 4
		}

		desc := make([]byte, align4(descsz))
		err = binary.Read(r, f.ByteOrder, &desc)
		if err != nil {
			return nil, err
		}

		d, err := parseNoteDesc(desc, addrsz, f.ByteOrder)
		if err != nil {
			return nil, err
		}

		note := Note{
			Location:  d.location,
			Base:      d.base,
			Semaphore: d.semaphore,
			Args:      d.args,
			bo:        f.ByteOrder,
		}

		if base != 0 {
			// From the SystemTap wiki about .stapsdt.base:
			//
			// Nothing about this section itself matters, we just use it as a marker to detect
			// prelink address adjustments.
			// Each probe note records the link-time address of the .stapsdt.base section alongside
			// the probe PC address. The decoder compares the base address stored in the note with
			// the .stapsdt.base section's sh_addr.
			// Initially these are the same, but the section header will be adjusted by prelink.
			// So the decoder applies the difference to the probe PC address to get the correct
			// prelinked PC address; the same adjustment is applied to the semaphore address, if any.
			diff := base - note.Base
			note.Location = offset(f, note.Location+diff)
			if note.Semaphore != 0 {
				note.Semaphore += diff
				note.SemaphoreOffsetRefctr = semOffset(f, note.Semaphore)
			}
		}

		if provider == d.provider && probe == d.probe {
			return &note, nil
		}
	}

	return nil, fmt.Errorf("%w: %s in provider %s", ErrProbeNotFound, probe, provider)
}

// noteDesc is the decoded body of one stapsdt note.
type noteDesc struct {
	location  uint64
	base      uint64
	semaphore uint64
	provider  string
	probe     string
	args      string
}

// parseNoteDesc decodes one stapsdt note descriptor: three addresses followed
// by the null-terminated provider, probe and argument strings. It returns an
// error rather than panicking on a descriptor too short for the addresses or
// carrying an unterminated provider or probe name.
func parseNoteDesc(desc []byte, addrsz int, bo binary.ByteOrder) (noteDesc, error) {
	if addrsz != 8 {
		return noteDesc{}, fmt.Errorf("unsupported address size: %d", addrsz)
	}

	if len(desc) < 3*addrsz {
		return noteDesc{}, fmt.Errorf("malformed stapsdt note: descriptor too short: %d", len(desc))
	}

	d := noteDesc{
		location:  bo.Uint64(desc[0:addrsz]),
		base:      bo.Uint64(desc[addrsz : 2*addrsz]),
		semaphore: bo.Uint64(desc[2*addrsz : 3*addrsz]),
	}

	idx := 3 * addrsz

	providersz := bytes.IndexByte(desc[idx:], 0)
	if providersz < 0 {
		return noteDesc{}, errors.New("malformed stapsdt note: unterminated provider name")
	}
	d.provider = string(desc[idx : idx+providersz])

	idx += providersz + 1
	probesz := bytes.IndexByte(desc[idx:], 0)
	if probesz < 0 {
		return noteDesc{}, errors.New("malformed stapsdt note: unterminated probe name")
	}
	d.probe = string(desc[idx : idx+probesz])

	// The arguments string follows immediately after the probe name's null terminator.
	idx += probesz + 1
	if idx < len(desc) {
		argssz := bytes.IndexByte(desc[idx:], 0)
		if argssz < 0 {
			argssz = len(desc) - idx
		}
		d.args = string(desc[idx : idx+argssz])
	}

	return d, nil
}

func offset(f *elf.File, addr uint64) uint64 {
	for _, prog := range f.Progs {
		if prog.Type != elf.PT_LOAD || (prog.Flags&elf.PF_X) == 0 {
			continue
		}
		if prog.Vaddr <= addr && addr < (prog.Vaddr+prog.Memsz) {
			return addr - prog.Vaddr + prog.Off
		}
	}
	return addr
}

func sdtBaseAddr(f *elf.File) uint64 {
	sec := f.Section(".stapsdt.base")
	if sec == nil {
		// .stapsdt.base not present
		return 0
	}
	return sec.Addr
}

func semOffset(f *elf.File, addr uint64) uint64 {
	sec := f.Section(".probes")
	if sec == nil {
		// .probes not present
		return addr
	}
	return addr - sec.Addr + sec.Offset
}
