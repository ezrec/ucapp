package sio

import (
	"fmt"
	"io"
	"io/fs"
	"iter"
	"log"
	"maps"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ezrec/ucapp/internal"
)

const (
	// DRUM_OP_MASK masks the drum operation type from an alert request.
	DRUM_OP_MASK = (1 << 8)
	// DRUM_OP_SELECT indicates a ring selection operation.
	DRUM_OP_SELECT = (0 << 8)
	// DRUM_OP_SELECT_MASK masks the ring ID from a select operation.
	DRUM_OP_SELECT_MASK = 0xff
	// DRUM_OP_RING indicates a ring-level operation.
	DRUM_OP_RING = (1 << 8)
)

var _drum_defines = map[string]string{
	"DRUM_OP_MASK":        fmt.Sprintf("0x%x", DRUM_OP_MASK),
	"DRUM_OP_SELECT":      fmt.Sprintf("0x%x", DRUM_OP_SELECT),
	"DRUM_OP_SELECT_MASK": fmt.Sprintf("0x%x", DRUM_OP_SELECT_MASK),
	"DRUM_OP_RING":        fmt.Sprintf("0x%x", DRUM_OP_RING),
}

// Drum represents a collection of up to 256 rings, providing persistent storage
// similar to a drum memory device. It implements the Channel interface by
// forwarding operations to the currently selected ring.
type Drum struct {
	*Ring
	Rings map[uint8](*Ring)
}

// Defines returns an iter of defines for the channel.
func (drum *Drum) Defines() iter.Seq2[string, string] {
	return internal.IterSeq2Concat(maps.All(_drum_defines), (&Ring{}).Defines())
}

// Rewind resets all rings in the drum to their initial positions.
func (drum *Drum) Rewind() {
	for _, ring := range drum.Rings {
		ring.Rewind()
	}
}

// Unmarshal loads drum data from a file system by scanning for ring files
// matching the pattern XX.ur (2 hex digits).
func (drum *Drum) Unmarshal(filesys fs.FS) (err error) {
	drum.Rings = map[uint8](*Ring){}

	return fs.WalkDir(filesys, ".", func(path string, d fs.DirEntry, err_in error) (err error) {
		if d.IsDir() {
			return
		}
		name := d.Name()
		ok, err := regexp.MatchString("(?i)[0-9a-f][0-9a-f].ur", name)
		if err != nil {
			return
		}
		if !ok {
			// Skip this file.
			return nil
		}
		ring_index, err := strconv.ParseUint(strings.TrimSuffix(name, filepath.Ext(name)), 16, 8)
		if err != nil {
			return
		}

		// Ensure the ring exists, and unmarshal it.
		ring := &Ring{}
		ring.Rewind()
		drum.Rings[uint8(ring_index)] = ring
		ring_io, err := filesys.Open(name)
		if err != nil {
			return
		}
		defer ring_io.Close()

		ring.Unmarshal(ring_io)

		return
	})
}

// Marshal writes the drum's rings to a file system, creating files named
// XX.ur for each ring.
func (drum *Drum) Marshal(filesys CreateFS) (err error) {
	for index, ring := range drum.Rings {
		if !ring.Dirty() {
			continue
		}

		ring_name := fmt.Sprintf("%02x.ur", index)
		var ring_file io.WriteCloser
		ring_file, err = filesys.Create(ring_name)
		if err != nil {
			return
		}

		err = ring.Marshal(ring_file)
		ring_file.Close()
		if err != nil {
			return
		}
	}

	return
}

// Receive returns an iterator that yields bits from the currently selected ring.
// If no ring is selected, selects ring 0 by default.
func (drum *Drum) Receive() iter.Seq[bool] {
	if drum == nil {
		return func(func(bool) bool) {}
	}

	if drum.Ring == nil {
		drum.selectRing(0)
	}

	return drum.Ring.Receive()
}

// Send writes a bit to the currently selected ring.
// If no ring is selected, selects ring 0 by default.
func (drum *Drum) Send(value bool) (err error) {
	if drum == nil {
		err = ErrDrumMissing
		return
	}

	if drum.Ring == nil {
		drum.selectRing(0)
	}

	err = drum.Ring.Send(value)
	return
}

// selectRing selectes a ring
func (drum *Drum) selectRing(selected uint8) {
	ring, ok := drum.Rings[selected]
	if !ok {
		if drum.Rings == nil {
			drum.Rings = make(map[uint8](*Ring))
		}
		ring = &Ring{}
		ring.Rewind()
		drum.Rings[selected] = ring
	}
	drum.Ring = ring
}

// Alert handles drum control operations including ring selection and
// forwarding ring-specific operations to the currently selected ring.
func (drum *Drum) Alert(request uint32, response chan uint32) {
	if drum == nil {
		response <- ^uint32(0)
		return
	}

	// Ring request
	switch request & DRUM_OP_MASK {
	case DRUM_OP_SELECT:
		selected := uint8(request & DRUM_OP_SELECT_MASK)
		drum.selectRing(selected)
		response <- uint32(len(drum.Ring.Data))
	case DRUM_OP_RING:
		drum.Ring.Alert(request, response)
	}
}

// Dirty returns true if any ring in the drum has unflushed changes.
func (drum *Drum) Dirty() bool {
	for _, ring := range drum.Rings {
		if ring.Dirty() {
			return true
		}
	}

	return false
}

// Delete a file from a drum.
func (drum *Drum) Delete(name string) (err error) {
	if drum.Rings == nil {
		drum.Rings = map[uint8](*Ring){}
	}

	ring_ff, ok := drum.Rings[0xff]
	if !ok {
		ring_ff = &Ring{}
		ring_ff.Rewind()
		drum.Rings[0xff] = ring_ff
	}

	var dd DrumDirent

	for n := 0; n < len(ring_ff.Data); n += dd.Size() {
		dd.Unmarshal(ring_ff.Data[n : n+dd.Size()])
		if dd.Deleted() {
			continue
		}

		if dd.NameIs(name) {
			dd.Delete()
			// Replace existing
			buff, _ := dd.Marshal()
			copy(ring_ff.Data[n:n+len(buff)], buff)
			ring_ff.isDirty = true
			break
		}
	}

	if !ring_ff.isDirty {
		err = fmt.Errorf("%v %w", name, fs.ErrNotExist)
	}

	return
}

// Save a file into a drum, allocating a name in the dirent for it.
func (drum *Drum) Save(name string, content io.Reader) (err error) {
	if drum.Rings == nil {
		drum.Rings = map[uint8](*Ring){}
	}

	ring_ff, ok := drum.Rings[0xff]
	if !ok {
		ring_ff = &Ring{}
		ring_ff.Rewind()
		drum.Rings[0xff] = ring_ff
	}
	// Create the allocation bitmap
	var alloc_map [4]uint64

	// Ring 0 and Ring 255 are always allocated
	alloc_map[0] |= 1
	alloc_map[3] |= 1 << 63

	dd := DrumDirent{
		Ring: 0xff,
	}

	dirent_offset := -1

	for n := 0; n < len(ring_ff.Data); n += dd.Size() {
		dd.Unmarshal(ring_ff.Data[n : n+dd.Size()])
		if dd.Deleted() {
			continue
		}

		if dd.NameIs(name) {
			// Use existing dirent, and ring.
			dirent_offset = n
			break
		}

		ring := dd.Ring
		alloc_map[ring/64] |= (1 << (ring % 64))
	}

	// No matching name - find first free dirent.
	if dirent_offset < 0 {
		for n := 0; n < len(ring_ff.Data); n += dd.Size() {
			de := DrumDirent{}
			de.Unmarshal(ring_ff.Data[n : n+dd.Size()])
			if de.Deleted() {
				dirent_offset = n
				break
			}
		}
		// Reset dd.Ring to indicate we need a new ring allocation
		dd.Ring = 0xff
	}

	if dd.Deleted() {
		// Find first unallocated
		for n, mask := range alloc_map {
			if mask != ^uint64(0) {
				for i := range 64 {
					if (mask & (1 << i)) == 0 {
						dd.Ring = uint8(n*64 + i)
						break
					}
				}
				break
			}
		}

		if dd.Ring == 0 {
			err = fmt.Errorf("unable to allocate ring")
			return
		}
	}

	dd.Name = name
	var buff []byte
	buff, err = dd.Marshal()
	if err != nil {
		return
	}

	if dirent_offset < 0 {
		ring_ff.Data = ring_ff.Data[:ring_ff.WriteIndex/8]
		ring_ff.Data = append(ring_ff.Data, buff...)
		ring_ff.isDirty = true
	} else {
		// Replace existing
		copy(ring_ff.Data[dirent_offset:dirent_offset+len(buff)], buff)
		ring_ff.isDirty = true
	}
	ring_ff.WriteIndex = len(ring_ff.Data) * 8

	ring, ok := drum.Rings[dd.Ring]
	if !ok {
		ring = &Ring{}
		drum.Rings[dd.Ring] = ring
	}

	ring.Data, err = io.ReadAll(content)
	if err != nil {
		return
	}

	ring.isDirty = true
	ring.ReadIndex = 0
	ring.WriteIndex = len(ring.Data) * 8

	return
}

// Get the sequence of directory entries from the drum.
func (drum *Drum) Dirents() iter.Seq[DrumDirent] {
	return func(yield func(dd DrumDirent) bool) {
		if drum.Rings == nil {
			return
		}

		ring_ff, ok := drum.Rings[0xff]
		if !ok {
			return
		}

		var dd DrumDirent
		for n := 0; n < len(ring_ff.Data); n += dd.Size() {
			dd.Unmarshal(ring_ff.Data[n : n+dd.Size()])
			if !yield(dd) {
				return
			}
		}
	}
}

// DrumDirent is a drum directory entry.
type DrumDirent struct {
	Name string
	Ring uint8
}

const (
	_depot_charset = "\0001234567890+-_.,@ABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// NameIs returns true if the name supplied would match the name in the dirent.
func (dirent *DrumDirent) NameIs(name string) bool {
	return strings.EqualFold(name, dirent.Name)
}

// Return the on-ring size of the dirent itself in bytes.
func (dirent *DrumDirent) Size() int {
	return 4
}

// Delete marks the entry as deleted
func (dirent *DrumDirent) Delete() {
	// Just set the ring to the same as the directory ring.
	dirent.Ring = 0xff
}

// Deleted returns true if the entry is deleted.
func (dirent *DrumDirent) Deleted() bool {
	// Is it marked as deleted?
	return dirent.Ring == 0xff
}

// Unmarshal converts a on-ring representation of the dirent into the structure.
func (dirent *DrumDirent) Unmarshal(content []uint8) (err error) {
	if len(content) < dirent.Size() {
		err = fmt.Errorf("drum dirent content is too small")
		return
	}

	dirent.Ring = content[3]
	var name_6 uint32
	name_6 = (uint32(content[0]) << 0) |
		(uint32(content[1]) << 8) |
		(uint32(content[2]) << 16)
	dirent.Name = ""
	for range 4 {
		chr := name_6 & 0x3f
		if chr == 0 {
			break
		}
		if int(chr) > len(_depot_charset) {
			log.Fatalf("unable to decode %d to ASCII", chr)
		}
		dirent.Name += string([]byte{_depot_charset[chr]})
		name_6 >>= 6
	}

	return
}

// Marshal converts the dirent to the on-ring resprentation
func (dirent *DrumDirent) Marshal() (content []uint8, err error) {
	content = make([]uint8, dirent.Size())
	content[3] = dirent.Ring

	if len(dirent.Name) == 0 {
		err = ErrNameTooShort
		return
	}

	if len(dirent.Name) > 4 {
		err = ErrNameTooLong
		return
	}

	uname := strings.ToUpper(dirent.Name)

	var name_6 uint32
	for n, letter := range uname {
		chr := strings.IndexRune(_depot_charset, letter)
		if chr < 0 {
			err = fmt.Errorf("unable to encode \"%s\" %w", dirent.Name, ErrNameRuneInvalid)
			return
		}
		name_6 |= uint32(chr) << (n * 6)
	}
	content[0] = uint8(name_6 >> 0)
	content[1] = uint8(name_6 >> 8)
	content[2] = uint8(name_6 >> 16)

	return
}
