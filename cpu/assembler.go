// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package cpu

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// Macro represents a macro definition in the assembly language.
type Macro struct {
	LineNo int      // Line number of the macro definition.
	Args   []string // Arguments for the macro.
	Lines  []string // Lines of macro text to expand.
}

// Predefined system equates
var sysEquate = map[string]string{
	"LINENO":     "0",
	"ARENA_MASK": fmt.Sprintf("%#v", ARENA_MASK),
	"ARENA_IO":   fmt.Sprintf("%#v", ARENA_IO),
	"ARENA_FREE": fmt.Sprintf("%#v", ARENA_FREE),
	"ARENA_TMP":  fmt.Sprintf("%#v", ARENA_TMP),
	"ARENA_CODE": fmt.Sprintf("%#v", ARENA_CODE),
}

// Assembler is a single pass macro assembler for the μCAPP system.
type Assembler struct {
	Verbose bool     // If set, verbosely logs the assembler actions.
	Opcode  []Opcode // List of generated opcodes.

	predefine map[string]string   // Predefines
	Label     map[string]int      // Map of jump labels to opcode indexes.
	Equate    map[string]string   // Map of equates.
	Macro     map[string](*Macro) // Map of macros.
}

// Define defines a new equate or redefines an existing equate.
func (asm *Assembler) Predefine(equ string, value string) {
	if asm.predefine == nil {
		asm.predefine = map[string]string{equ: value}
	} else {
		asm.predefine[equ] = value
	}
}

// dstMap is a map of register names to IR codes.
var dstMap = map[string]CodeIR{
	"r0":    IR_REG_R0,
	"r1":    IR_REG_R1,
	"r2":    IR_REG_R2,
	"r3":    IR_REG_R3,
	"r4":    IR_REG_R4,
	"r5":    IR_REG_R5,
	"ip":    IR_IP,
	"stack": IR_STACK,
}

// valueOf returns the value of a simple word.
func (asm *Assembler) valueOf(word string) (value uint32, err error) {
	invert := false
	if word[0] == '~' {
		invert = true
		word = word[1:]
	}
	if word[0] == '\'' {
		// Character quotes should have been expanded into
		// values in parseLine()
		err = ErrParseCharacter(word[1 : len(word)-1])
		return
	}
	v64, err := strconv.ParseInt(word, 0, 33)
	if err != nil {
		err = ErrParseNumber(word)
		return
	}

	if v64 <= 0xffffffff && v64 >= -int64(0x80000000) {
		if v64 < 0 {
			value = uint32(0xffffffff + (v64 + 1))
		} else {
			value = uint32(v64)
		}
	}

	if invert {
		value = ^value
	}

	return
}

// The 12 readable sources. Immediates are handled separately.
var irMap = map[string]CodeIR{
	"r0":    IR_REG_R0,
	"r1":    IR_REG_R1,
	"r2":    IR_REG_R2,
	"r3":    IR_REG_R3,
	"r4":    IR_REG_R4,
	"r5":    IR_REG_R5,
	"ip":    IR_IP,
	"stack": IR_STACK,
	"match": IR_REG_MATCH,
	"mask":  IR_REG_MASK,
	"first": IR_REG_FIRST,
	"count": IR_REG_COUNT,
}

// irOrImm determines if a value can be encoded as an CodeIR, or a set of immediates.
func (asm *Assembler) irOrImm(words ...string) (ir CodeIR, imms []uint16, err error) {
	if len(words) > 1 {
		err = ErrOpcodeExtraArgs
		return
	}

	if len(words) == 0 {
		ir = IR_CONST_FFFFFFFF
		return
	}

	word := words[0]

	ir, is_ir := irMap[word]
	if is_ir {
		// known value source - we're done!
		return
	}

	value, err := asm.valueOf(word)
	if err != nil {
		return
	}

	// Determine if encodable immediate, or if it
	// needs to be packed into the opcode words.
	switch {
	case value == 0:
		ir = IR_CONST_0
	case value == 0xffffffff:
		ir = IR_CONST_FFFFFFFF
	case value <= 0xffff:
		ir = IR_IMMEDIATE_16
		imms = []uint16{uint16((value >> 0) & 0xffff)}
	default:
		ir = IR_IMMEDIATE_32
		imms = []uint16{uint16((value >> 16) & 0xffff), uint16((value >> 0) & 0xffff)}
	}

	return
}

// parentEval does compile-time $(...) evaluations
func (asm *Assembler) parenEval(expr string) (value uint32, err error) {
	thread := starlark.Thread{}
	opts := syntax.FileOptions{}
	pred := starlark.StringDict{}
	for key, str := range asm.Equate {
		var value32 uint32
		value32, err = asm.valueOf(str)
		if err != nil {
			// Ignore non-integer equates. They may be registers
			// or something else.
			continue
		}
		pred[key] = starlark.MakeInt(int(value32))
	}
	prog := "rc=" + expr + "\n"
	dict, err := starlark.ExecFileOptions(&opts, &thread, "expr", prog, pred)
	if err != nil {
		return
	}
	st_rc, ok := dict["rc"]
	if !ok {
		err = ErrParseExpression(expr)
		return
	}
	st_int, ok := st_rc.(starlark.Int)
	if !ok {
		err = ErrParseExpression(expr)
		return
	}
	st_int64, ok := st_int.Int64()
	if !ok {
		err = ErrParseExpression(expr)
		return
	}
	value = uint32(st_int64)
	return
}

// parseLine parses a single line as an opcode.
func (asm *Assembler) parseLine(line string, lineno int) (words []string, err error) {
	// Set line number.
	asm.Equate["LINENO"] = fmt.Sprintf("%v", lineno)

	// Do 'x' evaluations
	re := regexp.MustCompile(`'\\?[^']'`)
	line = re.ReplaceAllStringFunc(line, func(word string) string {
		str := word[1 : len(word)-1]
		if str[0] == '\\' {
			str = str[1:]
			switch str {
			case "\\":
				str = "\\"
			case "n":
				str = "\n"
			case "r":
				str = "\r"
			case "e":
				str = "\033"
			default:
				return word
			}
		} else if len(str) != 1 {
			return word
		}
		return fmt.Sprintf("%v", str[0])
	})

	// Do $() evaluations
	re = regexp.MustCompile(`\$\([^\$]*\)`)
	line = re.ReplaceAllStringFunc(line, func(str string) string {
		value, _err := asm.parenEval(str[2 : len(str)-1])
		if _err != nil {
			err = _err
		}
		return fmt.Sprintf("%#v", value)
	})
	if err != nil {
		return
	}

	words = slices.DeleteFunc(strings.Split(line, " "), func(a string) bool { return len(a) == 0 })

	if len(words) == 0 {
		return
	}

	// .equ CONST VALUE
	if len(words) > 0 && words[0] == ".equ" {
		if len(words) != 3 {
			err = ErrEquateSyntax
			return
		}
		_, ok := asm.Equate[words[1]]
		if ok {
			err = ErrEquateDuplicate
			return
		}
		asm.Equate[words[1]] = words[2]
		words = words[:0]
		return
	}

	for n, word := range words {
		if len(word) == 0 {
			continue
		}

		// Check for equate next
		equate, ok := asm.Equate[word]
		if ok {
			words[n] = equate
		}
	}

	for strings.HasSuffix(words[0], ":") {
		label := words[0][:len(words[0])-1]
		_, ok := asm.Label[label]
		if ok {
			err = ErrLabelDuplicate
			return
		}

		if asm.Label == nil {
			asm.Label = make(map[string]int, 16)
		}
		asm.Label[label] = asm.currentIp()
		words = words[1:]
		if len(words) == 0 {
			return
		}
	}

	// .macro processing
	macro, ok := asm.Macro[words[0]]
	if ok {
		name := words[0]

		args := words[1:]
		if len(args) != len(macro.Args) {
			err = ErrMacroSyntax
			return
		}
		// Turn args into equs
		old_equate := maps.Clone(asm.Equate)
		for n, arg := range macro.Args {
			asm.Equate[arg] = words[1+n]
		}
		defer func() { asm.Equate = old_equate }()

		for n, line := range macro.Lines {
			lineno := macro.LineNo + n

			line = strings.ReplaceAll(line, "@", fmt.Sprintf("%v_%v_", name, lineno))
			words, err = asm.parseLine(line, lineno)
			if err != nil {
				err = &ErrMacro{Macro: name, Line: lineno, Err: err}
				err = &ErrSyntax{LineNo: lineno, Line: line, Err: err}
				return
			}

			err = asm.parseWords(words, macro.LineNo+n)
			if err != nil {
				err = &ErrMacro{Macro: name, Line: lineno, Err: err}
				err = &ErrSyntax{LineNo: lineno, Line: line, Err: err}
				return
			}
		}

		words = nil
		return
	}

	return
}

// currentIp gets the current Ip
func (asm *Assembler) currentIp() int {
	if len(asm.Opcode) == 0 {
		return 0
	}

	last := asm.Opcode[len(asm.Opcode)-1]

	return last.Ip + len(last.Codes)
}

// Parse parses an input stream into a Program containing opcodes.
func (asm *Assembler) Parse(input io.Reader) (prog *Program, err error) {

	scanner := bufio.NewScanner(input)

	var line string
	var lineno int
	var macro *Macro

	defer func() {
		if err != nil {
			err = &ErrSyntax{LineNo: lineno, Line: line, Err: err}
		}
	}()

	clear(asm.Label)
	asm.Opcode = asm.Opcode[:0]
	if asm.Macro == nil {
		asm.Macro = make(map[string](*Macro))
	}
	clear(asm.Macro)
	asm.Equate = maps.Clone(sysEquate)
	for attr, val := range asm.predefine {
		asm.Equate[attr] = val
	}

	for scanner.Scan() {
		text := scanner.Text()
		lineno += 1

		if asm.Verbose {
			log.Printf("%v: %v\n", lineno, text)
		}

		text_comment := strings.Split(text, ";")
		line = strings.TrimSpace(text_comment[0])
		all_words := strings.Split(line, " ")

		var words []string
		for _, single := range all_words {
			if len(single) > 0 {
				words = append(words, single)
			}
		}

		// .macro NAME arg...
		if len(words) > 0 && words[0] == ".macro" {
			if macro != nil {
				err = ErrMacroNesting
				return
			}
			_, ok := asm.Macro[words[1]]
			if ok {
				err = ErrMacroDuplicate
				return
			}
			macro = &Macro{
				LineNo: lineno + 1,
			}
			if len(words) > 2 {
				macro.Args = words[2:]
			}
			asm.Macro[words[1]] = macro
			continue
		}

		if len(words) > 0 && words[0] == ".endm" {
			if macro == nil {
				err = ErrMacroLonelyEndm
				return
			}
			macro = nil
			continue
		}

		if macro != nil {
			macro.Lines = append(macro.Lines, line)
			continue
		}

		words, err = asm.parseLine(line, lineno)
		if err != nil {
			return
		}

		err = asm.parseWords(words, lineno)
		if err != nil {
			return
		}
	}

	if macro != nil {
		err = ErrMacroLonely
		return
	}

	// Final linking of jump labels.
	for n := range asm.Opcode {
		op := &asm.Opcode[n]

		if len(op.LinkLabel) == 0 {
			continue
		}
		label := op.LinkLabel
		ip, ok := asm.Label[label]
		if !ok {
			err = ErrLabelMissing(label)
			return
		}
		if len(op.Codes) < 1 {
			log.Fatalf("Unable to link label '%s' to line %d: %v", label, op.LineNo, op.Words)
		}
		linked := &op.Codes[len(op.Codes)-1]
		if len(linked.Immediates) < 2 {
			log.Fatalf("Missing immediates for link label '%s' at line %d: %v", label, op.LineNo, op.Words)
		}
		linked.Immediates[0] |= uint16((ip >> 16) & 0xffff)
		linked.Immediates[1] |= uint16((ip >> 0) & 0xffff)
	}

	prog = &Program{
		Opcodes: slices.Clone(asm.Opcode),
	}

	return
}

// aluMap maps ALU opcode names.
var aluMap = map[string]CodeAluOp{
	"set": ALU_OP_SET,
	"xor": ALU_OP_XOR,
	"and": ALU_OP_AND,
	"or":  ALU_OP_OR,
	"shl": ALU_OP_SHL,
	"shr": ALU_OP_SHR,
	"add": ALU_OP_ADD,
	"sub": ALU_OP_SUB,
}

// channelMap maps IO channel names.
var channelMap = map[string]CodeChannel{
	"temp":    CHANNEL_ID_TEMP,
	"depot":   CHANNEL_ID_DEPOT,
	"tape":    CHANNEL_ID_TAPE,
	"vt":      CHANNEL_ID_VT,
	"monitor": CHANNEL_ID_MONITOR,
}

// getChannel gets the channel code for a word.
func (asm *Assembler) getChannel(word string) (channel CodeChannel, err error) {
	channel, ok := channelMap[word]
	if ok {
		return
	}
	value, err := asm.valueOf(word)
	if err != nil {
		return
	}

	if value > 8 {
		err = ErrChannelInvalid
		return
	}

	channel = CodeChannel(value)

	return
}

// getMatchMask returns the match & mask encoding
func (asm *Assembler) getMatchMask(cond CodeCond, words []string) (match, mask CodeIR, imms []uint16, err error) {
	if len(words) > 2 {
		err = ErrOpcodeExtraArgs
		return
	}
	if len(words) == 0 {
		match = IR_CONST_0
		mask = IR_CONST_FFFFFFFF
		return
	}

	if len(words) == 1 {
		words = append(words, "0xffffffff")
	}

	out := [2](*CodeIR){&match, &mask}
	for n, word := range words {
		var ir CodeIR
		var ir_imms []uint16
		ir, ir_imms, err = asm.irOrImm(word)
		if err != nil {
			return
		}
		*out[n] = ir
		imms = append(imms, ir_imms...)
	}

	return
}

// parseWords evaluates the words in a line of assembly text.
func (asm *Assembler) parseWords(words []string, lineno int) (err error) {
	var codes []Code
	var label string

	// no-op
	if len(words) == 0 {
		return
	}

	initial_words := words

	defer func() {
		if len(codes) == 0 {
			return
		}
		opcode := Opcode{LineNo: lineno, Ip: asm.currentIp(), Words: initial_words, Codes: codes, LinkLabel: label}
		asm.Opcode = append(asm.Opcode, opcode)
	}()

	cond := COND_ALWAYS

	switch words[0] {
	case "?":
		cond = COND_TRUE
		words = words[1:]
	case "!":
		cond = COND_FALSE
		words = words[1:]
	}

	var word_is_dst bool
	if len(words) >= 2 {
		_, word_is_dst = dstMap[words[1]]
	}

	// Alternate syntax substitutions
	switch {
	case len(words) >= 2 && words[0] == "write" && words[1] == "list":
		// write list VALUE MASK => list write VALUE MASK
		words[0] = "list"
		words[1] = "write"
	case len(words) >= 2 && words[0] == "write" && words[1] == "first":
		// write first VALUE MASK => list first VALUE MASK
		words[0] = "list"
		words[1] = "first"
	case len(words) >= 2 && words[0] == "write" && word_is_dst:
		// write <dst> VALUE MASK => alu set <dst> VALUE MASK
		words = append([]string{"alu", "set"}, words[1:]...)
	case len(words) == 2 && words[0] == "if" && words[1] == "some?":
		// if some? => if gt? count 0
		words = []string{"if", "gt?", "count", "0"}
	case len(words) == 2 && words[0] == "if" && words[1] == "none?":
		// if none? => if eq? count 0
		words = []string{"if", "eq?", "count", "0"}
	case len(words) == 3 && words[0] == "if" && words[1] == "true?":
		// if true? SRCA => if ne? SRCA 0
		words = []string{"if", "ne?", words[2], "0"}
	case len(words) == 3 && words[0] == "if" && words[1] == "false?":
		// if false? SRCA => if eq? SRCA 0
		words = []string{"if", "eq?", words[2], "0"}
	case len(words) == 1 && words[0] == "trap":
		// trap -> io await monitor
		words = []string{"io", "await", "monitor"}
	case len(words) >= 1 && words[0] == "fetch":
		words = append([]string{"io"}, words...)
	case len(words) >= 1 && words[0] == "store":
		words = append([]string{"io"}, words...)
	case len(words) >= 1 && words[0] == "await":
		words = append([]string{"io"}, words...)
	case len(words) >= 1 && words[0] == "alert":
		words = append([]string{"io"}, words...)
	case len(words) == 1 && words[0] == "return":
		words = []string{"alu", "set", "ip", "stack"}
	case len(words) == 2 && words[0] == "vjump":
		words = []string{"alu", "set", "ip", words[1]}
	default:
		// unchanged
	}

	switch words[0] {
	case "if":
		if len(words) < 2 {
			err = ErrOpcodeMissing
			return
		}

		var imms []uint16
		a := IR_CONST_0
		b := IR_CONST_0
		op := COND_OP_EQ
		if len(words) < 4 {
			err = ErrOpcodeValueMissing
			return
		}
		if len(words) > 4 {
			err = ErrOpcodeExtraArgs
			return
		}
		a, b, imms, err = asm.getMatchMask(cond, words[2:])
		switch words[1] {
		case "eq?":
			op = COND_OP_EQ
		case "ne?":
			op = COND_OP_NE
		case "le?":
			op = COND_OP_LE
		case "lt?":
			op = COND_OP_LT
		case "ge?":
			op = COND_OP_LT
			b, a = a, b
		case "gt?":
			op = COND_OP_LE
			b, a = a, b
		default:
			err = ErrOpcodeInvalid
			return
		}
		codes = append(codes, MakeCodeCond(cond, op, a, b, imms...))
	case "list":
		if len(words) < 2 {
			err = ErrOpcodeMissing
			return
		}
		var imms []uint16
		switch words[1] {
		case "all":
			if len(words) > 2 {
				err = ErrOpcodeExtraArgs
				return
			}
			codes = append(codes, MakeCodeCapp(cond, CAPP_OP_LIST_ALL, IR_CONST_0, IR_CONST_0))
		case "not":
			if len(words) > 2 {
				err = ErrOpcodeExtraArgs
				return
			}
			codes = append(codes, MakeCodeCapp(cond, CAPP_OP_LIST_NOT, IR_CONST_0, IR_CONST_0))
		case "next":
			if len(words) > 2 {
				err = ErrOpcodeExtraArgs
				return
			}
			codes = append(codes, MakeCodeCapp(cond, CAPP_OP_LIST_NEXT, IR_CONST_0, IR_CONST_0))
		case "of":
			if len(words) < 3 {
				err = ErrOpcodeValueMissing
				return
			}
			var match, mask CodeIR
			match, mask, imms, err = asm.getMatchMask(cond, words[2:])
			if err != nil {
				return
			}
			codes = append(codes, MakeCodeCapp(cond, CAPP_OP_SET_OF, match, mask, imms...))
		case "only":
			var match, mask CodeIR
			if len(words) < 3 {
				err = ErrOpcodeValueMissing
				return
			}
			match, mask, imms, err = asm.getMatchMask(cond, words[2:])
			if err != nil {
				return
			}
			codes = append(codes, MakeCodeCapp(cond, CAPP_OP_LIST_ONLY, match, mask, imms...))
		case "write":
			var match, mask CodeIR
			if len(words) < 3 {
				err = ErrOpcodeValueMissing
				return
			}
			match, mask, imms, err = asm.getMatchMask(cond, words[2:])
			if err != nil {
				return
			}
			codes = append(codes, MakeCodeCapp(cond, CAPP_OP_WRITE_LIST, match, mask, imms...))
		case "first":
			var match, mask CodeIR
			if len(words) < 3 {
				err = ErrOpcodeValueMissing
				return
			}
			match, mask, imms, err = asm.getMatchMask(cond, words[2:])
			if err != nil {
				return
			}
			codes = append(codes, MakeCodeCapp(cond, CAPP_OP_WRITE_FIRST, match, mask, imms...))
		default:
			err = ErrOpcodeInvalid
			return
		}
	case "io":
		if len(words) < 3 {
			err = ErrOpcodeMissing
			return
		}
		var channel CodeChannel
		channel, err = asm.getChannel(words[2])
		if err != nil {
			return
		}
		var arg CodeIR
		var imms []uint16
		arg, imms, err = asm.irOrImm(words[3:]...)
		if err != nil {
			return
		}
		switch words[1] {
		case "fetch":
			codes = append(codes, MakeCodeIo(cond, IO_OP_FETCH, channel, arg, imms...))
		case "store":
			codes = append(codes, MakeCodeIo(cond, IO_OP_STORE, channel, arg, imms...))
		case "alert":
			codes = append(codes, MakeCodeIo(cond, IO_OP_ALERT, channel, arg, imms...))
		case "await":
			if arg != IR_CONST_FFFFFFFF && !arg.Writable() {
				err = ErrOpcodeInvalid
				return
			}
			codes = append(codes, MakeCodeIo(cond, IO_OP_AWAIT, channel, arg, imms...))
		default:
			err = ErrOpcodeInvalid
			return
		}
	case "call":
		if len(words) < 2 {
			err = ErrOpcodeMissing
			return
		}
		if len(words) > 2 {
			err = ErrOpcodeExtraArgs
			return
		}
		codes = append(codes,
			MakeCodeAlu(cond, ALU_OP_SET, IR_STACK, IR_IMMEDIATE_16, 1),
			MakeCodeAlu(cond, ALU_OP_ADD, IR_STACK, IR_IP),
			MakeCodeAlu(cond, ALU_OP_SET, IR_IP, IR_IMMEDIATE_32, 0, 0),
		)
		label = words[1]
	case "vcall":
		if len(words) < 2 {
			err = ErrOpcodeMissing
			return
		}
		var imms []uint16
		var arg CodeIR
		arg, imms, err = asm.irOrImm(words[1:]...)
		if err != nil {
			return
		}
		codes = append(codes,
			MakeCodeAlu(cond, ALU_OP_SET, IR_STACK, IR_IMMEDIATE_16, 1),
			MakeCodeAlu(cond, ALU_OP_ADD, IR_STACK, IR_IP),
			MakeCodeAlu(cond, ALU_OP_SET, IR_IP, arg, imms...),
		)
	case "return":
		if len(words) > 1 {
			err = ErrOpcodeExtraArgs
			return
		}
		codes = append(codes,
			MakeCodeAlu(cond, ALU_OP_SET, IR_IP, IR_STACK),
		)
	case "exit":
		codes = append(codes, MakeCodeExit(cond))
	case "jump":
		if len(words) < 2 {
			err = ErrOpcodeMissing
			return
		}
		if len(words) > 2 {
			err = ErrOpcodeExtraArgs
			return
		}
		codes = append(codes,
			MakeCodeAlu(cond, ALU_OP_SET, IR_IP, IR_IMMEDIATE_32, 0, 0),
		)
		label = words[1]
	case "alu":
		if len(words) < 4 {
			err = ErrOpcodeMissing
			return
		}
		if len(words) > 4 {
			err = ErrOpcodeExtraArgs
			return
		}
		alu, ok := aluMap[words[1]]
		if !ok {
			err = ErrOpcodeInvalid
			return
		}
		reg, ok := dstMap[words[2]]
		if !ok {
			err = ErrTargetInvalid
			return
		}
		var arg CodeIR
		var imms []uint16
		arg, imms, err = asm.irOrImm(words[3:]...)
		if err != nil {
			return
		}
		codes = append(codes, MakeCodeAlu(cond, alu, reg, arg, imms...))
	default:
		err = ErrInstructionInvalid
		return
	}

	return
}
