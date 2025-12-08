package cpu

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Unit tests for Assembler multi-parse functionality.
//
// These tests validate the ability to call Parse() multiple times between
// Clear() and Link(), which allows assembling multiple source files into
// a single program while preserving labels, equates, and macros across files.
//
// KEY FINDINGS:
//
// 1. CORRECT BEHAVIOR:
//    - Clear() initializes the assembler and sets ready=true
//    - Multiple Parse() calls accumulate opcodes with continuous IP values
//    - Labels defined in one Parse() can be referenced in another
//    - Equates and macros defined in one Parse() are available in subsequent Parse() calls
//    - Link() resolves all forward references and produces final program
//    - Duplicate labels, equates, and macros are detected across Parse() calls
//    - Filename tracking works correctly with os.File's Name() method
//
// 3. RECOMMENDATION:
//    Add a check at the start of Parse() to enforce ready==true:
//        if !asm.ready {
//            return ErrNotReady
//        }
//    This would prevent all three bug scenarios above.

// TestAssemblerMultiParse validates calling Parse() multiple times between Clear() and Link().
func TestAssemblerMultiParse(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	// Parse first file
	prog1 := []string{
		"write r0 0x10",
		"write r1 0x20",
	}
	err := asm.Parse(strings.NewReader(strings.Join(prog1, "\n")))
	assert.NoError(err)

	// Parse second file
	prog2 := []string{
		"write r2 0x30",
		"write r3 0x40",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog2, "\n")))
	assert.NoError(err)

	// Link
	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)

	// Should have all 4 opcodes
	expected := []Opcode{
		{"stdin", 1, 0, []string{"write", "r0", "0x10"}, []Code{
			MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x10)}, ""},
		{"stdin", 2, 1, []string{"write", "r1", "0x20"}, []Code{
			MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R1, IR_IMMEDIATE_16, 0x20)}, ""},
		{"stdin", 1, 2, []string{"write", "r2", "0x30"}, []Code{
			MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R2, IR_IMMEDIATE_16, 0x30)}, ""},
		{"stdin", 2, 3, []string{"write", "r3", "0x40"}, []Code{
			MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R3, IR_IMMEDIATE_16, 0x40)}, ""},
	}

	opEqual(t, expected, prog.Opcodes)
}

// TestAssemblerMultiParseWithLabels validates that labels work across multiple Parse() calls.
func TestAssemblerMultiParseWithLabels(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	// First parse: forward reference to LABEL2
	prog1 := []string{
		"LABEL1:",
		"write r0 0x10",
		"jump LABEL2",
	}
	err := asm.Parse(strings.NewReader(strings.Join(prog1, "\n")))
	assert.NoError(err)

	// Second parse: define LABEL2 and jump back to LABEL1
	prog2 := []string{
		"LABEL2:",
		"write r1 0x20",
		"jump LABEL1",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog2, "\n")))
	assert.NoError(err)

	// Link should resolve both labels
	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)

	// Verify opcodes: write r0, jump LABEL2, write r1, jump LABEL1
	assert.Equal(4, len(prog.Opcodes))

	// Check that jumps were linked correctly
	// Jump to LABEL2 should point to IP 2
	jumpToLabel2 := prog.Opcodes[1]
	assert.Equal("LABEL2", jumpToLabel2.LinkLabel)
	assert.Equal(1, len(jumpToLabel2.Codes))
	assert.Equal(2, len(jumpToLabel2.Codes[0].Immediates))
	linkedIp2 := uint32(jumpToLabel2.Codes[0].Immediates[0])<<16 | uint32(jumpToLabel2.Codes[0].Immediates[1])
	assert.Equal(uint32(2), linkedIp2)

	// Jump to LABEL1 should point to IP 0
	jumpToLabel1 := prog.Opcodes[3]
	assert.Equal("LABEL1", jumpToLabel1.LinkLabel)
	assert.Equal(1, len(jumpToLabel1.Codes))
	assert.Equal(2, len(jumpToLabel1.Codes[0].Immediates))
	linkedIp1 := uint32(jumpToLabel1.Codes[0].Immediates[0])<<16 | uint32(jumpToLabel1.Codes[0].Immediates[1])
	assert.Equal(uint32(0), linkedIp1)
}

// TestAssemblerMultiParseWithEquates validates that equates defined in one Parse() are available in subsequent Parse() calls.
func TestAssemblerMultiParseWithEquates(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	// First parse: define equates
	prog1 := []string{
		".equ CONST_A 0x100",
		".equ CONST_B 0x200",
		"write r0 CONST_A",
	}
	err := asm.Parse(strings.NewReader(strings.Join(prog1, "\n")))
	assert.NoError(err)

	// Second parse: use equates from first parse
	prog2 := []string{
		"write r1 CONST_B",
		"write r2 $(CONST_A + CONST_B)",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog2, "\n")))
	assert.NoError(err)

	// Link
	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)

	expected := []Opcode{
		{"stdin", 3, 0, []string{"write", "r0", "0x100"}, []Code{
			MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x100)}, ""},
		{"stdin", 1, 1, []string{"write", "r1", "0x200"}, []Code{
			MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R1, IR_IMMEDIATE_16, 0x200)}, ""},
		{"stdin", 2, 2, []string{"write", "r2", "0x300"}, []Code{
			MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R2, IR_IMMEDIATE_16, 0x300)}, ""},
	}

	opEqual(t, expected, prog.Opcodes)
}

// TestAssemblerMultiParseWithMacros validates that macros defined in one Parse() are available in subsequent Parse() calls.
func TestAssemblerMultiParseWithMacros(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	// First parse: define macro
	prog1 := []string{
		".macro SETVAL rn val",
		"write rn val",
		".endm",
		"SETVAL r0 0x10",
	}
	err := asm.Parse(strings.NewReader(strings.Join(prog1, "\n")))
	assert.NoError(err)

	// Second parse: use macro from first parse
	prog2 := []string{
		"SETVAL r1 0x20",
		"SETVAL r2 0x30",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog2, "\n")))
	assert.NoError(err)

	// Link
	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)

	expected := []Opcode{
		{"stdin", 2, 0, []string{"write", "r0", "0x10"}, []Code{
			MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x10)}, ""},
		{"stdin", 2, 1, []string{"write", "r1", "0x20"}, []Code{
			MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R1, IR_IMMEDIATE_16, 0x20)}, ""},
		{"stdin", 2, 2, []string{"write", "r2", "0x30"}, []Code{
			MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R2, IR_IMMEDIATE_16, 0x30)}, ""},
	}

	opEqual(t, expected, prog.Opcodes)
}

// TestAssemblerMultiParseDuplicateLabel validates that duplicate labels across Parse() calls are detected.
func TestAssemblerMultiParseDuplicateLabel(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	// First parse: define LABEL1
	prog1 := []string{
		"LABEL1:",
		"write r0 0x10",
	}
	err := asm.Parse(strings.NewReader(strings.Join(prog1, "\n")))
	assert.NoError(err)

	// Second parse: try to redefine LABEL1 (should fail)
	prog2 := []string{
		"LABEL1:",
		"write r1 0x20",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog2, "\n")))
	assert.Error(err)
	assert.ErrorIs(err, ErrLabelDuplicate)
}

// TestAssemblerMultiParseDuplicateEquate validates that duplicate equates across Parse() calls are detected.
func TestAssemblerMultiParseDuplicateEquate(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	// First parse: define CONST_A
	prog1 := []string{
		".equ CONST_A 0x100",
	}
	err := asm.Parse(strings.NewReader(strings.Join(prog1, "\n")))
	assert.NoError(err)

	// Second parse: try to redefine CONST_A (should fail)
	prog2 := []string{
		".equ CONST_A 0x200",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog2, "\n")))
	assert.Error(err)
	assert.ErrorIs(err, ErrEquateDuplicate)
}

// TestAssemblerMultiParseDuplicateMacro validates that duplicate macros across Parse() calls are detected.
func TestAssemblerMultiParseDuplicateMacro(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	// First parse: define macro
	prog1 := []string{
		".macro MYMAC val",
		"write r0 val",
		".endm",
	}
	err := asm.Parse(strings.NewReader(strings.Join(prog1, "\n")))
	assert.NoError(err)

	// Second parse: try to redefine MYMAC (should fail)
	prog2 := []string{
		".macro MYMAC val",
		"write r1 val",
		".endm",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog2, "\n")))
	assert.Error(err)
	assert.ErrorIs(err, ErrMacroDuplicate)
}

// TestAssemblerParseWithoutClear validates that Parse() requires Clear() to be called first.
func TestAssemblerParseWithoutClear(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}

	// Try to parse without calling Clear() first
	prog := []string{
		"write r0 0x10",
	}

	// This currently panics with "assignment to entry in nil map"
	// because Equate map is nil
	defer func() {
		if r := recover(); r != nil {
			// Expected to panic due to uninitialized maps
			assert.Contains(r, "nil map", "Expected panic due to nil map")
		}
	}()

	err := asm.Parse(strings.NewReader(strings.Join(prog, "\n")))
	assert.Error(err)
}

// TestAssemblerParseThenLink validates behavior when Parse() is called after Link().
func TestAssemblerParseThenLink(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	// First parse
	prog1 := []string{
		"write r0 0x10",
	}
	err := asm.Parse(strings.NewReader(strings.Join(prog1, "\n")))
	assert.NoError(err)

	// Link (sets ready = false)
	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)
	assert.Equal(1, len(prog.Opcodes))

	// Try to parse after link (ready flag is false)
	prog2 := []string{
		"write r1 0x20",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog2, "\n")))
	assert.Error(err)

	// If we link again, we get both opcodes
	_, err = asm.Link()
	assert.Error(err)
}

// TestAssemblerMultiParseErrorRecovery validates that an error in one Parse() sets ready=false.
// CURRENT BEHAVIOR: After an error, ready=false but Parse() doesn't check it.
func TestAssemblerMultiParseErrorRecovery(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	// First parse: valid
	prog1 := []string{
		"write r0 0x10",
	}
	err := asm.Parse(strings.NewReader(strings.Join(prog1, "\n")))
	assert.NoError(err)

	// Second parse: invalid syntax (should set ready=false)
	prog2 := []string{
		"invalid_instruction",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog2, "\n")))
	assert.Error(err)

	// After error, ready should be false
	assert.False(asm.ready, "ready flag should be false after parse error")

	prog3 := []string{
		"write r1 0x20",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog3, "\n")))
	assert.Error(err)

	// Cannot link after corrupted state
	_, err = asm.Link()
	assert.Error(err)
}

// TestAssemblerMultiParseWithFiles validates proper filename tracking across multiple Parse() calls.
func TestAssemblerMultiParseWithFiles(t *testing.T) {
	assert := assert.New(t)

	// Create temporary files
	file1, err := os.CreateTemp("", "test1_*.asm")
	assert.NoError(err)
	defer os.Remove(file1.Name())
	_, err = file1.WriteString("write r0 0x10\n")
	assert.NoError(err)
	file1.Close()

	file2, err := os.CreateTemp("", "test2_*.asm")
	assert.NoError(err)
	defer os.Remove(file2.Name())
	_, err = file2.WriteString("write r1 0x20\n")
	assert.NoError(err)
	file2.Close()

	asm := &Assembler{}
	asm.Clear()

	// Parse first file
	f1, err := os.Open(file1.Name())
	assert.NoError(err)
	err = asm.Parse(f1)
	f1.Close()
	assert.NoError(err)

	// Parse second file
	f2, err := os.Open(file2.Name())
	assert.NoError(err)
	err = asm.Parse(f2)
	f2.Close()
	assert.NoError(err)

	// Link
	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)

	// Verify filenames are preserved
	assert.Equal(2, len(prog.Opcodes))
	assert.Contains(prog.Opcodes[0].Filename, "test1_")
	assert.Contains(prog.Opcodes[1].Filename, "test2_")
}

// TestAssemblerMultiParseEmptyFiles validates that empty Parse() calls are handled correctly.
func TestAssemblerMultiParseEmptyFiles(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	// Parse empty content
	err := asm.Parse(strings.NewReader(""))
	assert.NoError(err)

	// Parse more empty content
	err = asm.Parse(strings.NewReader("\n\n"))
	assert.NoError(err)

	// Parse actual content
	prog1 := []string{
		"write r0 0x10",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog1, "\n")))
	assert.NoError(err)

	// Parse empty again
	err = asm.Parse(strings.NewReader(""))
	assert.NoError(err)

	// Link
	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)

	// Should have exactly 1 opcode
	assert.Equal(1, len(prog.Opcodes))
}

// TestAssemblerMultiParseIPContinuity validates that IP values continue correctly across Parse() calls.
func TestAssemblerMultiParseIPContinuity(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	// First parse: 2 opcodes (IP 0, 1)
	prog1 := []string{
		"write r0 0x10",
		"write r1 0x20",
	}
	err := asm.Parse(strings.NewReader(strings.Join(prog1, "\n")))
	assert.NoError(err)

	// Second parse: 2 opcodes (IP should be 2, 3)
	prog2 := []string{
		"write r2 0x30",
		"write r3 0x40",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog2, "\n")))
	assert.NoError(err)

	// Link
	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)

	// Verify IP continuity
	assert.Equal(4, len(prog.Opcodes))
	assert.Equal(0, prog.Opcodes[0].Ip)
	assert.Equal(1, prog.Opcodes[1].Ip)
	assert.Equal(2, prog.Opcodes[2].Ip)
	assert.Equal(3, prog.Opcodes[3].Ip)
}

// TestAssemblerMultiParsePredefine validates that predefines work across multiple Parse() calls.
func TestAssemblerMultiParsePredefine(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Predefine("MY_CONST", "0x999")
	asm.Clear()

	// First parse: use predefined value
	prog1 := []string{
		"write r0 MY_CONST",
	}
	err := asm.Parse(strings.NewReader(strings.Join(prog1, "\n")))
	assert.NoError(err)

	// Second parse: use predefined value again
	prog2 := []string{
		"write r1 MY_CONST",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog2, "\n")))
	assert.NoError(err)

	// Link
	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)

	expected := []Opcode{
		{"stdin", 1, 0, []string{"write", "r0", "0x999"}, []Code{
			MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x999)}, ""},
		{"stdin", 1, 1, []string{"write", "r1", "0x999"}, []Code{
			MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R1, IR_IMMEDIATE_16, 0x999)}, ""},
	}

	opEqual(t, expected, prog.Opcodes)
}

// TestAssemblerMultiParseMacroWithFilename validates that macro filenames are tracked correctly.
func TestAssemblerMultiParseMacroWithFilename(t *testing.T) {
	assert := assert.New(t)

	// Create temporary file for macro definition
	file1, err := os.CreateTemp("", "macros_*.asm")
	assert.NoError(err)
	defer os.Remove(file1.Name())
	_, err = file1.WriteString(".macro SETVAL rn val\nwrite rn val\n.endm\n")
	assert.NoError(err)
	file1.Close()

	asm := &Assembler{}
	asm.Clear()

	// Parse macro definition from file
	f1, err := os.Open(file1.Name())
	assert.NoError(err)
	err = asm.Parse(f1)
	f1.Close()
	assert.NoError(err)

	// Parse usage from stdin
	prog2 := []string{
		"SETVAL r0 0x10",
	}
	err = asm.Parse(strings.NewReader(strings.Join(prog2, "\n")))
	assert.NoError(err)

	// Link
	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)

	// The expanded macro line should reference the original file where the macro was defined
	assert.Equal(1, len(prog.Opcodes))
	assert.Contains(prog.Opcodes[0].Filename, "macros_")
}

// TestAssemblerMultiParseComplexScenario validates a complex multi-file assembly scenario.
func TestAssemblerMultiParseComplexScenario(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	// File 1: Constants and macros
	file1 := []string{
		".equ BASE_ADDR 0x1000",
		".macro LOAD reg offset",
		"write reg $(BASE_ADDR + offset)",
		".endm",
	}
	err := asm.Parse(strings.NewReader(strings.Join(file1, "\n")))
	assert.NoError(err)

	// File 2: Main code with forward reference
	file2 := []string{
		"START:",
		"LOAD r0 0x10",
		"jump END",
	}
	err = asm.Parse(strings.NewReader(strings.Join(file2, "\n")))
	assert.NoError(err)

	// File 3: More code
	file3 := []string{
		"LOAD r1 0x20",
		"END:",
		"exit",
	}
	err = asm.Parse(strings.NewReader(strings.Join(file3, "\n")))
	assert.NoError(err)

	// Link
	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)

	// Should have 4 opcodes: LOAD r0 (from macro), jump, LOAD r1 (from macro), exit
	assert.Equal(4, len(prog.Opcodes))

	// Verify jump was linked correctly to END (IP 3)
	jumpOp := prog.Opcodes[1]
	assert.Equal("END", jumpOp.LinkLabel)
	linkedIp := uint32(jumpOp.Codes[0].Immediates[0])<<16 | uint32(jumpOp.Codes[0].Immediates[1])
	assert.Equal(uint32(3), linkedIp)
}
