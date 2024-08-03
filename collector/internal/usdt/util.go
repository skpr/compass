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

type Note struct {
	Location              uint64
	Base                  uint64
	Semaphore             uint64
	SemaphoreOffsetPtrace uint64
	SemaphoreOffsetRefctr uint64
	bo                    binary.ByteOrder
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

	addrsz := 4
	if f.Class == elf.ELFCLASS64 {
		addrsz = 8
	}

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

		err = binary.Read(r, f.ByteOrder, &descsz)
		if err != nil {
			return nil, err
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

		note := Note{
			Location:  f.ByteOrder.Uint64(desc[0:addrsz]),
			Base:      f.ByteOrder.Uint64(desc[addrsz : 2*addrsz]),
			Semaphore: f.ByteOrder.Uint64(desc[2*addrsz : 3*addrsz]),
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

		idx := 3 * addrsz
		providersz := bytes.IndexByte(desc[idx:], 0)
		pv := string(desc[idx : idx+providersz])

		idx += providersz + 1
		probesz := bytes.IndexByte(desc[idx:], 0)
		pb := string(desc[idx : idx+probesz])

		if provider == pv && probe == pb {
			return &note, nil
		}
	}

	return nil, fmt.Errorf("probe %s not found in provider %s", probe, provider)
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
