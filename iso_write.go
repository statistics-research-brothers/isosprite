package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"
)

const sectorSize = 2048

func putBoth32(b []byte, v uint32) {
	binary.LittleEndian.PutUint32(b[0:4], v)
	binary.BigEndian.PutUint32(b[4:8], v)
}

func putBoth16(b []byte, v uint16) {
	binary.LittleEndian.PutUint16(b[0:2], v)
	binary.BigEndian.PutUint16(b[2:4], v)
}

func padBytes(s string, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	copy(b, s)
	return b
}

func dirRecordLen(idLen int) int {
	l := 33 + idLen
	if idLen%2 == 0 {
		l++
	}
	return l
}

func buildDirRecord(id []byte, isDir bool, lba uint32, size uint32, dt [7]byte) []byte {
	idLen := len(id)
	rl := dirRecordLen(idLen)
	buf := make([]byte, rl)
	buf[0] = byte(rl)
	putBoth32(buf[2:10], lba)
	putBoth32(buf[10:18], size)
	copy(buf[18:25], dt[:])
	if isDir {
		buf[25] = 0x02
	}
	putBoth16(buf[28:32], 1)
	buf[32] = byte(idLen)
	copy(buf[33:33+idLen], id)
	return buf
}

func recordingDateTime() [7]byte {
	var b [7]byte
	now := time.Now()
	b[0] = byte(now.Year() - 1900)
	b[1] = byte(now.Month())
	b[2] = byte(now.Day())
	b[3] = byte(now.Hour())
	b[4] = byte(now.Minute())
	b[5] = byte(now.Second())
	return b
}

func packEntries(entries [][]byte) []byte {
	var out []byte
	sector := make([]byte, sectorSize)
	cur := 0
	for _, e := range entries {
		if cur+len(e) > sectorSize {
			out = append(out, sector...)
			sector = make([]byte, sectorSize)
			cur = 0
		}
		copy(sector[cur:], e)
		cur += len(e)
	}
	out = append(out, sector...)
	return out
}

func buildDirectoryExtent(d *isoNode, parent *isoNode, dt [7]byte) []byte {
	selfLBA, selfSize := d.lba, d.sectors*sectorSize
	var parentLBA, parentSize uint32
	if parent != nil {
		parentLBA, parentSize = parent.lba, parent.sectors*sectorSize
	} else {
		parentLBA, parentSize = selfLBA, selfSize
	}
	entries := [][]byte{
		buildDirRecord([]byte{0x00}, true, selfLBA, selfSize, dt),
		buildDirRecord([]byte{0x01}, true, parentLBA, parentSize, dt),
	}
	for _, c := range d.children {
		id := []byte(c.isoID)
		var lba, size uint32
		if c.isDir {
			lba, size = c.lba, c.sectors*sectorSize
		} else {
			lba, size = c.lba, uint32(c.size)
		}
		entries = append(entries, buildDirRecord(id, c.isDir, lba, size, dt))
	}
	return packEntries(entries)
}

func collectDirsBFS(root *isoNode) []*isoNode {
	root.pathNum = 1
	order := []*isoNode{root}
	queue := []*isoNode{root}
	num := 1
	for len(queue) > 0 {
		d := queue[0]
		queue = queue[1:]
		for _, c := range d.children {
			if c.isDir {
				num++
				c.pathNum = num
				c.parentNum = d.pathNum
				order = append(order, c)
				queue = append(queue, c)
			}
		}
	}
	return order
}

func collectFilesDFS(d *isoNode, out *[]*isoNode) {
	for _, c := range d.children {
		if c.isDir {
			collectFilesDFS(c, out)
		} else {
			*out = append(*out, c)
		}
	}
}

func buildPathTableRecord(id []byte, lba uint32, parentNum int, bigEndian bool) []byte {
	idLen := len(id)
	rl := 8 + idLen
	if idLen%2 != 0 {
		rl++
	}
	buf := make([]byte, rl)
	buf[0] = byte(idLen)
	if bigEndian {
		binary.BigEndian.PutUint32(buf[2:6], lba)
		binary.BigEndian.PutUint16(buf[6:8], uint16(parentNum))
	} else {
		binary.LittleEndian.PutUint32(buf[2:6], lba)
		binary.LittleEndian.PutUint16(buf[6:8], uint16(parentNum))
	}
	copy(buf[8:8+idLen], id)
	return buf
}

func buildPathTable(dirs []*isoNode, bigEndian bool) []byte {
	var out []byte
	for _, d := range dirs {
		var id []byte
		if d.pathNum == 1 {
			id = []byte{0x00}
		} else {
			id = []byte(d.isoID)
		}
		out = append(out, buildPathTableRecord(id, d.lba, d.parentNum, bigEndian)...)
	}
	return out
}

func ceilSectors(n int) uint32 {
	return uint32((n + sectorSize - 1) / sectorSize)
}

func isoDateTime17() []byte {
	now := time.Now()
	s := fmt.Sprintf("%04d%02d%02d%02d%02d%02d00", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	b := make([]byte, 17)
	copy(b, []byte(s))
	return b
}

func baseName(p string) string {
	last := 0
	for i := 0; i < len(p); i++ {
		if p[i] == '/' {
			last = i + 1
		}
	}
	name := p[last:]
	if len(name) > 4 && name[len(name)-4:] == ".iso" {
		name = name[:len(name)-4]
	}
	return name
}

func CreateISO(srcDir string, outPath string) error {
	info, err := os.Stat(srcDir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", srcDir)
	}

	root, err := buildTree(srcDir, "")
	if err != nil {
		return err
	}

	dt := recordingDateTime()

	dirs := collectDirsBFS(root)
	root.parentNum = 1
	for _, d := range dirs {
		probe := buildDirectoryExtent(d, nil, dt)
		d.sectors = uint32(len(probe) / sectorSize)
	}

	var files []*isoNode
	collectFilesDFS(root, &files)

	pathTableL := buildPathTable(dirs, false)
	pathTableM := buildPathTable(dirs, true)
	pathTableSize := len(pathTableL)
	pathTableSectors := ceilSectors(pathTableSize)

	lba := uint32(18)
	pathTableL_LBA := lba
	lba += pathTableSectors
	pathTableM_LBA := lba
	lba += pathTableSectors

	for _, d := range dirs {
		d.lba = lba
		lba += d.sectors
	}
	for _, fl := range files {
		fl.lba = lba
		sec := ceilSectors(int(fl.size))
		if sec == 0 {
			sec = 1
		}
		fl.sectors = sec
		lba += sec
	}

	totalSectors := lba

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if err := out.Truncate(int64(totalSectors) * sectorSize); err != nil {
		return err
	}

	parentOf := map[*isoNode]*isoNode{}
	var mapParents func(d *isoNode)
	mapParents = func(d *isoNode) {
		for _, c := range d.children {
			if c.isDir {
				parentOf[c] = d
				mapParents(c)
			}
		}
	}
	mapParents(root)

	writeAt := func(offset int64, data []byte) error {
		_, err := out.WriteAt(data, offset)
		return err
	}

	for _, d := range dirs {
		data := buildDirectoryExtent(d, parentOf[d], dt)
		if err := writeAt(int64(d.lba)*sectorSize, data); err != nil {
			return err
		}
	}

	if err := writeAt(int64(pathTableL_LBA)*sectorSize, pathTableL); err != nil {
		return err
	}
	if err := writeAt(int64(pathTableM_LBA)*sectorSize, pathTableM); err != nil {
		return err
	}

	for _, fl := range files {
		content, err := os.ReadFile(fl.srcPath)
		if err != nil {
			return err
		}
		if err := writeAt(int64(fl.lba)*sectorSize, content); err != nil {
			return err
		}
	}

	pvd := make([]byte, sectorSize)
	pvd[0] = 1
	copy(pvd[1:6], []byte("CD001"))
	pvd[6] = 1
	copy(pvd[8:40], padBytes("", 32))
	volID := sanitizeName(baseName(outPath), true)
	copy(pvd[40:72], padBytes(volID, 32))
	putBoth32(pvd[80:88], totalSectors)
	putBoth16(pvd[120:124], 1)
	putBoth16(pvd[124:128], 1)
	putBoth16(pvd[128:132], sectorSize)
	putBoth32(pvd[132:140], uint32(pathTableSize))
	binary.LittleEndian.PutUint32(pvd[140:144], pathTableL_LBA)
	binary.BigEndian.PutUint32(pvd[148:152], pathTableM_LBA)
	rootRec := buildDirRecord([]byte{0x00}, true, root.lba, root.sectors*sectorSize, dt)
	copy(pvd[156:190], rootRec)
	copy(pvd[190:318], padBytes("", 128))
	copy(pvd[318:446], padBytes("", 128))
	copy(pvd[446:574], padBytes("", 128))
	copy(pvd[574:702], padBytes("", 128))
	copy(pvd[702:739], padBytes("", 37))
	copy(pvd[739:776], padBytes("", 37))
	copy(pvd[776:813], padBytes("", 37))
	copy(pvd[813:830], isoDateTime17())
	copy(pvd[830:847], isoDateTime17())
	copy(pvd[864:881], isoDateTime17())
	pvd[881] = 1

	if err := writeAt(16*sectorSize, pvd); err != nil {
		return err
	}

	term := make([]byte, sectorSize)
	term[0] = 255
	copy(term[1:6], []byte("CD001"))
	term[6] = 1
	if err := writeAt(17*sectorSize, term); err != nil {
		return err
	}

	return nil
}
