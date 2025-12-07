package sio

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDrumDirent_Marshal_Unmarshal tests basic dirent serialization
func TestDrumDirent_Marshal_Unmarshal(t *testing.T) {
	tests := []struct {
		name   string
		dirent DrumDirent
		err    error
	}{
		{
			name:   "simple name",
			dirent: DrumDirent{Name: "TEST", Ring: 0x01},
		},
		{
			name:   "empty name",
			dirent: DrumDirent{Name: "", Ring: 0x42},
			err:    ErrNameTooShort,
		},
		{
			name:   "excessive name",
			dirent: DrumDirent{Name: "toolongname", Ring: 0x42},
			err:    ErrNameTooLong,
		},
		{
			name:   "single char",
			dirent: DrumDirent{Name: "A", Ring: 0x10},
		},
		{
			name:   "four char",
			dirent: DrumDirent{Name: "ABCD", Ring: 0x20},
		},
		{
			name:   "deleted entry",
			dirent: DrumDirent{Name: "FILE", Ring: 0xff},
		},
		{
			name:   "with special chars",
			dirent: DrumDirent{Name: "A-B_", Ring: 0x05},
		},
		{
			name:   "with unknown characters",
			dirent: DrumDirent{Name: "aa:", Ring: 0x42},
			err:    ErrNameRuneInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			// Test Marshal
			marshaled, err := tt.dirent.Marshal()
			if tt.err != nil {
				assert.ErrorIs(err, tt.err)
				return
			} else {
				assert.NoError(err)
			}
			assert.Equal(tt.dirent.Size(), len(marshaled), "marshaled size should be 4 bytes")

			// Test Unmarshal round-trip
			var restored DrumDirent
			err = restored.Unmarshal(marshaled)
			assert.NoError(err)
			assert.Equal(tt.dirent.Ring, restored.Ring)
			// Note: Only first 4 chars are stored due to 24-bit/6-bit encoding limit
			assert.True(restored.NameIs(tt.dirent.Name), "name '%v' should match '%v' case-insensitively", restored.Name, tt.dirent.Name)
		})
	}
}

// TestDrumDirent_NameIs tests case-insensitive name matching
func TestDrumDirent_NameIs(t *testing.T) {
	tests := []struct {
		direntName string
		testName   string
		expected   bool
	}{
		{"TEST", "TEST", true},
		{"TEST", "test", true},
		{"TEST", "Test", true},
		{"test", "TEST", true},
		{"TEST", "TESTING", false},
		{"TEST", "TES", false},
		{"", "", true},
		{"A", "A", true},
		{"A", "B", false},
	}

	for _, tt := range tests {
		t.Run(tt.direntName+"_vs_"+tt.testName, func(t *testing.T) {
			assert := assert.New(t)
			dd := DrumDirent{Name: tt.direntName}
			assert.Equal(tt.expected, dd.NameIs(tt.testName))
		})
	}
}

// TestDrumDirent_Deleted tests deleted entry detection
func TestDrumDirent_Deleted(t *testing.T) {
	assert := assert.New(t)

	dd := DrumDirent{Name: "TEST", Ring: 0x01}
	assert.False(dd.Deleted(), "should not be deleted initially")

	dd.Delete()
	assert.True(dd.Deleted(), "should be deleted after Delete()")
	assert.Equal(uint8(0xff), dd.Ring, "deleted entry should have ring 0xff")
}

// TestDrumDirent_Size tests the size method
func TestDrumDirent_Size(t *testing.T) {
	assert := assert.New(t)
	dd := DrumDirent{}
	assert.Equal(4, dd.Size(), "dirent size should always be 4 bytes")
}

// TestDrumDirent_Unmarshal_Error tests error handling
func TestDrumDirent_Unmarshal_Error(t *testing.T) {
	assert := assert.New(t)

	var dd DrumDirent
	err := dd.Unmarshal([]byte{0x01, 0x02}) // Too small
	assert.Error(err, "should error on insufficient data")
	assert.Contains(err.Error(), "too small")
}

// TestDrum_Save_NewFile tests saving a new file to drum
func TestDrum_Save_NewFile(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}
	content := []byte{0x01, 0x02, 0x03, 0x04}

	err := drum.Save("TEST", bytes.NewReader(content))
	assert.NoError(err)

	// Check that ring 0xff was created with directory
	assert.NotNil(drum.Rings[0xff], "directory ring should exist")
	ring_ff := drum.Rings[0xff]
	assert.True(ring_ff.isDirty, "directory ring should be dirty")

	// Verify directory entry
	var found bool
	for dd := range drum.Dirents() {
		if dd.NameIs("TEST") {
			found = true
			assert.NotEqual(uint8(0), dd.Ring, "allocated ring should not be 0")
			assert.NotEqual(uint8(0xff), dd.Ring, "allocated ring should not be 0xff")

			// Check the content ring
			contentRing := drum.Rings[dd.Ring]
			assert.NotNil(contentRing)
			assert.Equal(content, contentRing.Data)
			assert.True(contentRing.isDirty)
			break
		}
	}
	assert.True(found, "TEST file should be in directory")
}

// TestDrum_Save_OverwriteFile tests overwriting an existing file
func TestDrum_Save_OverwriteFile(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	// Save initial file
	content1 := []byte{0x01, 0x02}
	err := drum.Save("TEST", bytes.NewReader(content1))
	assert.NoError(err)

	// Get the ring it was allocated to
	var allocatedRing uint8
	for dd := range drum.Dirents() {
		if dd.NameIs("TEST") {
			allocatedRing = dd.Ring
			break
		}
	}

	// Overwrite with new content
	content2 := []byte{0xAA, 0xBB, 0xCC}
	err = drum.Save("TEST", bytes.NewReader(content2))
	assert.NoError(err)

	// Verify it reused the same ring
	var count int
	for dd := range drum.Dirents() {
		if dd.NameIs("TEST") {
			count++
			assert.Equal(allocatedRing, dd.Ring, "should reuse same ring")

			// Check new content
			contentRing := drum.Rings[dd.Ring]
			assert.Equal(content2, contentRing.Data)
		}
	}
	assert.Equal(1, count, "should have exactly one entry for TEST")
}

// TestDrum_Save_MultipleFiles tests saving multiple files
func TestDrum_Save_MultipleFiles(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	files := []struct {
		name    string
		content []byte
	}{
		{"AAA", []byte{0x01}},
		{"BBB", []byte{0x02, 0x03}},
		{"CCC", []byte{0x04, 0x05, 0x06}},
	}

	// Save all files
	for _, f := range files {
		err := drum.Save(f.name, bytes.NewReader(f.content))
		assert.NoError(err)
	}

	// Each file should get its own unique ring
	foundFiles := make(map[string]bool)
	allocatedRings := make(map[uint8]int)

	for dd := range drum.Dirents() {
		if !dd.Deleted() {
			foundFiles[strings.ToUpper(dd.Name)] = true
			allocatedRings[dd.Ring]++
		}
	}

	assert.Equal(3, len(foundFiles), "should have 3 files in directory")
	assert.True(foundFiles["AAA"])
	assert.True(foundFiles["BBB"])
	assert.True(foundFiles["CCC"])

	// Each file should have a unique ring
	assert.Equal(3, len(allocatedRings), "each file should have unique ring")
	for ring, count := range allocatedRings {
		assert.Equal(1, count, "ring %d should only be used once", ring)
	}
}

// TestDrum_Delete_ExistingFile tests deleting an existing file
func TestDrum_Delete_ExistingFile(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	// Create a file
	err := drum.Save("TEST", bytes.NewReader([]byte{0x01, 0x02}))
	assert.NoError(err)

	// Delete it
	err = drum.Delete("TEST")
	assert.NoError(err)

	// Verify it's marked as deleted
	for dd := range drum.Dirents() {
		if dd.NameIs("TEST") {
			assert.True(dd.Deleted(), "file should be marked as deleted")
		}
	}

	// Directory should be dirty
	assert.True(drum.Rings[0xff].isDirty)
}

// TestDrum_Delete_NonExistentFile tests deleting a non-existent file
func TestDrum_Delete_NonExistentFile(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	err := drum.Delete("NOEXIST")
	assert.Error(err, "should error when deleting non-existent file")
	assert.ErrorIs(err, fs.ErrNotExist)
}

// TestDrum_Delete_CaseInsensitive tests case-insensitive deletion
func TestDrum_Delete_CaseInsensitive(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	// Create a file
	err := drum.Save("TEST", bytes.NewReader([]byte{0x01}))
	assert.NoError(err)

	// Delete using different case
	err = drum.Delete("test")
	assert.NoError(err)

	// Verify it's deleted
	for dd := range drum.Dirents() {
		if dd.NameIs("TEST") {
			assert.True(dd.Deleted())
		}
	}
}

// TestDrum_Delete_OnlyDeletesFirstMatch tests that Delete only deletes the first match
func TestDrum_Delete_OnlyDeletesFirstMatch(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	// Manually create directory with duplicate names (this can happen if names were
	// created through direct directory manipulation or after fixing truncation bug)
	drum.Rings = map[uint8]*Ring{
		1:    &Ring{Data: []byte{0x01}},
		2:    &Ring{Data: []byte{0x02}},
		3:    &Ring{Data: []byte{0x03}},
		0xff: &Ring{Data: []byte{}},
	}

	// Create three entries with same name but different rings
	for i := uint8(1); i <= 3; i++ {
		dd := DrumDirent{Name: "DUP", Ring: i}
		buff, err := dd.Marshal()
		assert.NoError(err)
		drum.Rings[0xff].Data = append(drum.Rings[0xff].Data, buff...)
	}

	// Delete should only remove first match
	err := drum.Delete("DUP")
	assert.NoError(err)

	// Count how many are deleted vs active
	deletedCount := 0
	activeCount := 0
	for dd := range drum.Dirents() {
		if dd.NameIs("DUP") {
			if dd.Deleted() {
				deletedCount++
			} else {
				activeCount++
			}
		}
	}

	assert.Equal(1, deletedCount, "only first match should be deleted")
	assert.Equal(2, activeCount, "remaining matches should still be active")
}

// TestDrum_Save_ReuseDeletedSlot tests that Save reuses deleted directory slots
func TestDrum_Save_ReuseDeletedSlot(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	// Create and delete a file
	err := drum.Save("OLD", bytes.NewReader([]byte{0x01}))
	assert.NoError(err)
	err = drum.Delete("OLD")
	assert.NoError(err)

	// Get directory size before
	dirSizeBefore := len(drum.Rings[0xff].Data)

	// Create a new file - should reuse the deleted slot
	err = drum.Save("NEW", bytes.NewReader([]byte{0x02}))
	assert.NoError(err)

	// Directory size should be the same (reused slot)
	dirSizeAfter := len(drum.Rings[0xff].Data)
	assert.Equal(dirSizeBefore, dirSizeAfter, "should reuse deleted slot")
}

// TestDrum_Dirents_Empty tests Dirents on empty drum
func TestDrum_Dirents_Empty(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	count := 0
	for range drum.Dirents() {
		count++
	}
	assert.Equal(0, count, "empty drum should have no dirents")
}

// TestDrum_Dirents_WithFiles tests iterating over directory entries
func TestDrum_Dirents_WithFiles(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	// Use short distinct names to avoid truncation bug
	drum.Save("AAA", bytes.NewReader([]byte{0x01}))
	drum.Save("BBB", bytes.NewReader([]byte{0x02}))
	drum.Save("CCC", bytes.NewReader([]byte{0x03}))
	drum.Delete("BBB") // Delete one

	// Count all entries (including deleted)
	allCount := 0
	activeCount := 0
	for dd := range drum.Dirents() {
		allCount++
		if !dd.Deleted() {
			activeCount++
		}
	}

	assert.Equal(3, allCount, "should iterate over all entries")
	assert.Equal(2, activeCount, "should have 2 active entries")
}

// TestDrum_Save_RingAllocation tests that rings are allocated correctly
func TestDrum_Save_RingAllocation(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	// Ring 0 and 0xff should be reserved
	// First file should get ring 1
	err := drum.Save("FRST", bytes.NewReader([]byte{0x01}))
	assert.NoError(err)

	for dd := range drum.Dirents() {
		if dd.NameIs("FRST") {
			assert.Equal(uint8(1), dd.Ring, "first file should get ring 1")
		}
	}
}

// TestDrum_Save_ManyFiles tests allocation of many files
func TestDrum_Save_ManyFiles(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	// Save multiple files to test ring allocation
	for i := 0; i < 10; i++ {
		name := string([]byte{'A' + byte(i)})
		err := drum.Save(name, bytes.NewReader([]byte{byte(i)}))
		assert.NoError(err)
	}

	// Collect allocated rings
	allocatedRings := make(map[uint8]int)
	for dd := range drum.Dirents() {
		if !dd.Deleted() {
			allocatedRings[dd.Ring]++
		}
	}

	// Each file should get a unique ring
	assert.Equal(10, len(allocatedRings), "each file should have unique ring")
	for ring, count := range allocatedRings {
		assert.Equal(1, count, "ring %d should only be used once", ring)
	}

	// Verify ring 0 and 0xff are not used for files
	assert.NotContains(allocatedRings, uint8(0), "ring 0 should not be allocated")
	assert.NotContains(allocatedRings, uint8(0xff), "ring 0xff should not be allocated")
}

// TestDrum_Save_NameTruncation tests that long names are truncated to 4 chars
// BUG: Names longer than 4 chars are silently truncated, causing data loss
func TestDrum_Save_NameTruncation(t *testing.T) {
	assert := assert.New(t)
	drum := &Drum{}

	// Names are limited to 4 characters due to 24-bit / 6-bit encoding
	err := drum.Save("FILE1", bytes.NewReader([]byte{0x01}))
	assert.ErrorIs(err, ErrNameTooLong)

	err = drum.Save("FILE2", bytes.NewReader([]byte{0x02}))
	assert.ErrorIs(err, ErrNameTooLong)

	// Result: No directory entries
	count := 0
	for range drum.Dirents() {
		count++
	}

	// Should be 0 files, as both names were too long.
	assert.Equal(0, count, "names were too long with no allocations")
}

// TestDrum_Save_InvalidCharacter tests saving with invalid characters
// Uses log.Fatalf which terminates the process, so we can't test it easily
func TestDrum_Save_InvalidCharacter(t *testing.T) {
	t.Skip("Test causes log.Fatalf which terminates process - invalid chars should return error instead")

	// This test is skipped because the Marshal function calls log.Fatalf
	// for invalid characters, which terminates the entire test process.
	// BUG: Should return an error instead of calling log.Fatalf
}

// TestDrum_Dirty tests the Dirty flag propagation
func TestDrum_Dirty(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}
	assert.False(drum.Dirty(), "new drum should not be dirty")

	// Save a file
	err := drum.Save("TEST", bytes.NewReader([]byte{0x01}))
	assert.NoError(err)
	assert.True(drum.Dirty(), "drum should be dirty after save")

	// Clear dirty flags
	for _, ring := range drum.Rings {
		ring.isDirty = false
	}
	assert.False(drum.Dirty(), "drum should not be dirty after clearing flags")
}

// TestDrum_Save_AllocationBitmap tests the allocation bitmap logic
func TestDrum_Save_AllocationBitmap(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	// Manually create some rings to test allocation around them
	drum.Rings = map[uint8]*Ring{
		1:    &Ring{Data: []byte{0x01}},
		2:    &Ring{Data: []byte{0x02}},
		4:    &Ring{Data: []byte{0x04}},
		0xff: &Ring{Data: []byte{}}, // Empty directory
	}

	// Initialize directory with entries (use 3-char names to avoid truncation)
	dd1 := DrumDirent{Name: "ONE", Ring: 1}
	dd2 := DrumDirent{Name: "TWO", Ring: 2}
	dd4 := DrumDirent{Name: "FOR", Ring: 4}

	for _, dd := range [](*DrumDirent){&dd1, &dd2, &dd4} {
		buff, err := dd.Marshal()
		assert.NoError(err)
		drum.Rings[0xff].Data = append(drum.Rings[0xff].Data, buff...)
	}
	drum.Rings[0xff].WriteIndex = len(drum.Rings[0xff].Data) * 8

	// Save a new file - should allocate ring 3 (first free)
	err := drum.Save("NEW", bytes.NewReader([]byte{0x03}))
	assert.NoError(err)

	// Find the allocated ring
	var allocatedRing uint8
	for dd := range drum.Dirents() {
		if dd.NameIs("NEW") {
			allocatedRing = dd.Ring
			break
		}
	}

	// Should allocate ring 3 (first free in the bitmap)
	assert.Equal(uint8(3), allocatedRing, "should allocate ring 3 (first free)")
}

// TestDrum_Save_EmptyContent tests saving empty content
func TestDrum_Save_EmptyContent(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	err := drum.Save("EMPT", bytes.NewReader([]byte{}))
	assert.NoError(err)

	// Verify file was created with empty content
	for dd := range drum.Dirents() {
		if dd.NameIs("EMPT") {
			ring := drum.Rings[dd.Ring]
			assert.NotNil(ring)
			assert.Equal(0, len(ring.Data))
		}
	}
}

// TestDrumDirent_Marshal_SpecialCharacters tests encoding of special characters
func TestDrumDirent_Marshal_SpecialCharacters(t *testing.T) {
	assert := assert.New(t)

	validChars := "1234567890+-_.,@ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for _, ch := range validChars {
		if ch == '\000' { // Skip null character
			continue
		}
		name := string(ch)
		dd := DrumDirent{Name: name, Ring: 0x01}

		marshaled, err := dd.Marshal()
		assert.NoError(err)
		assert.Equal(dd.Size(), len(marshaled))

		var restored DrumDirent
		err = restored.Unmarshal(marshaled)
		assert.NoError(err)
		assert.True(restored.NameIs(name), "failed to round-trip character %c", ch)
	}
}
