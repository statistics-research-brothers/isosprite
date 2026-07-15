package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
)

func ExtractISO(isoPath string, destDir string) error {
	f, err := os.Open(isoPath)
	if err != nil {
		return err
	}
	defer f.Close()

	pvd := make([]byte, sectorSize)
	if _, err := f.ReadAt(pvd, 16*sectorSize); err != nil {
		return err
	}

	rootRec := pvd[156:190]
	rootLBA := binary.LittleEndian.Uint32(rootRec[2:6])
	rootSize := binary.LittleEndian.Uint32(rootRec[10:14])

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	return extractDirectory(f, rootLBA, rootSize, destDir)
}

func extractDirectory(f *os.File, lba uint32, size uint32, destDir string) error {
	buf := make([]byte, size)
	if _, err := f.ReadAt(buf, int64(lba)*sectorSize); err != nil {
		return err
	}
	off := 0
	for off < len(buf) {
		rl := int(buf[off])
		if rl == 0 {
			next := ((off / sectorSize) + 1) * sectorSize
			if next <= off {
				break
			}
			off = next
			continue
		}
		rec := buf[off : off+rl]
		extentLBA := binary.LittleEndian.Uint32(rec[2:6])
		dataLen := binary.LittleEndian.Uint32(rec[10:14])
		flags := rec[25]
		idLen := int(rec[32])
		id := rec[33 : 33+idLen]
		off += rl
		if idLen == 1 && (id[0] == 0x00 || id[0] == 0x01) {
			continue
		}
		name := string(id)
		if i := strings.Index(name, ";"); i >= 0 {
			name = name[:i]
		}
		isDir := flags&0x02 != 0
		target := filepath.Join(destDir, name)
		if isDir {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			if err := extractDirectory(f, extentLBA, dataLen, target); err != nil {
				return err
			}
		} else {
			content := make([]byte, dataLen)
			if _, err := f.ReadAt(content, int64(extentLBA)*sectorSize); err != nil {
				return err
			}
			if err := os.WriteFile(target, content, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
