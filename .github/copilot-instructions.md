# μCAPP Copilot Instructions

## Project Overview

μCAPP (micro Content Addressable Parallel Processor) is a Go implementation of an abstract processor based on concepts from Caxton C. Foster's 1976 book "Content Addressable Parallel Processors". This project simulates a unique computer architecture where memory is addressed by content rather than by index, creating a parallel processing system with content-addressable memory (CAM).

## Architecture Components

- **CAPP (Content Addressable Parallel Processor)**: Main memory system implementing content-addressable parallel operations
- **CPU/Microprocessor**: Instruction pointer, six 32-bit registers, ALU, conditional execution flags, and stack
- **I/O System**: Multiple channels including Temp, Depot (drum storage), Tape, VT (Virtual Terminal), Monitor/ROM
- **Emulator**: Main emulator combining CPU, CAPP, and I/O subsystems
- **Assembler**: Custom assembler for the μCAPP assembly language
- **Translation**: Internationalization support using golang.org/x/text

## Project Structure

```
/cmd/ucapp/        - Command-line interface and main entry point
/capp/             - Content Addressable Parallel Processor implementation
/cpu/              - CPU, assembler, opcodes, and instruction execution
/emulator/         - High-level emulator combining all components
/io/               - I/O peripherals (Tape, Depot, Drum, Ring, VT, ROM, etc.)
/os/               - Operating system components
/translate/        - Internationalization and localization
/examples/         - Example programs (io, logic, math)
/drat/             - Additional tooling
```

## Code Standards

### Documentation Requirements

All new Go code **must** follow [Go documentation standards](https://go.dev/doc/effective_go#commentary):

1. **Package Documentation**: Every package must have a package comment. For multi-file packages, it should appear in one file (typically `doc.go` or the main package file).
   ```go
   // Package capp implements a Content Addressable Parallel Processor.
   // It provides operations for content-based memory addressing and
   // parallel data manipulation.
   package capp
   ```

2. **Exported Identifiers**: Every exported function, type, constant, and variable must have a doc comment starting with the identifier's name.
   ```go
   // NewCapp creates a new CAPP with the specified number of cells.
   func NewCapp(count uint) *Capp { ... }
   
   // Cell represents an individual CAPP memory cell with tag bits,
   // data storage, and set membership information.
   type Cell struct { ... }
   ```

3. **Doc Comment Format**:
   - Start with the identifier name
   - Use complete sentences
   - First sentence should be a summary (appears in package listings)
   - Additional paragraphs for detailed explanation when needed

4. **Internal Functions**: Document complex internal (unexported) functions to aid maintainability, but focus documentation efforts on exported APIs.

### Go Conventions

- Follow standard Go formatting (`gofmt`, `goimports`)
- Use meaningful variable names; avoid unnecessary abbreviations
- Prefer table-driven tests for comprehensive test coverage
- Use `go generate` directives where appropriate (e.g., `stringer` for enum types)
- Keep functions focused and concise; extract complex logic into separate functions

### Tools in Use

The project uses these Go tools (defined in `go.mod`):
- `golang.org/x/text/cmd/gotext` - Internationalization
- `golang.org/x/tools/cmd/stringer` - Generate String() methods for enums
- `honnef.co/go/tools/cmd/staticcheck` - Static analysis

Run these tools before committing:
```bash
go generate ./...
go tool staticcheck ./...
go test ./...
```

## Domain-Specific Considerations

### CAPP Architecture
- Memory cells are addressed by content, not index
- Operations work on tagged subsets of cells in parallel
- Cell structure includes Set bits, Tag bit, Data (32-bit), and Next pointer
- The BitsFlipped counter and SetsSwapped flag track CAPP state

### CPU Architecture
- 6 registers (r0-r5), each 32-bit
- Instruction Pointer (IP) has three modes: CAPP, Stack, and Register execution
- Match/Mask registers control CAPP cell selection
- Conditional execution based on Cond flag
- Power and Ticks track resource usage

### Assembly Language
- Custom assembly language defined in `cpu/ASSEMBLY.md`
- Opcodes documented in `cpu/OPCODES.md`
- Instructions target CAPP operations, ALU operations, I/O, and conditionals
- Immediate values and register references

### I/O Channels (0-7)
- **0**: Temp - Temporary storage
- **1**: Depot - Persistent drum/ring storage
- **2**: Tape - Sequential I/O
- **3**: VT - Virtual Terminal (128x64 text display)
- **6**: Debug channel
- **7**: ROM/Monitor - Boot ROM and inter-process communication

### File Formats
- **Ring**: `\xb5RNG` magic - Sequential byte storage, up to 64KB
- **Drum**: `\xb5DRM` magic - Collection of up to 256 rings
- **Tape**: Raw byte streams for sequential I/O

## Testing Strategy

- Unit tests for individual components (`*_test.go`)
- Fuzz tests for CPU instructions (`cpu_fuzz_test.go`)
- Integration tests through the emulator
- Example programs serve as integration tests

When adding features:
1. Write or update unit tests first
2. Ensure existing tests continue to pass
3. Add example programs demonstrating new functionality
4. Update relevant README.md files in affected packages

## Historical Context

This project explores a "retro-future" where STARAN-like content-addressable processing technology (developed for the US Air Force in the 1970s) became accessible to hobbyists like the Intel 8080 or MOS 6502 did in actual history. Understanding this context helps inform design decisions that maintain the vintage computing aesthetic while implementing in modern Go.

## Common Tasks

### Adding a New Opcode
1. Define opcode in `cpu/opcode.go`
2. Update `cpu/OPCODES.md` documentation
3. Implement execution logic in CPU
4. Add test cases in `cpu/cpu_test.go`
5. Update `cpu/ASSEMBLY.md` if assembly syntax changes
6. Create example program demonstrating the opcode

### Adding a New I/O Channel
1. Define interface implementation in `io/`
2. Add channel constant in `cpu/cpu.go`
3. Update `io/README.md` with channel documentation
4. Wire up in `emulator/emulator.go`
5. Add test coverage

### Modifying CAPP Behavior
1. Update `capp/capp.go` with new operations
2. Ensure Cell structure changes are documented
3. Update tests in `capp/capp_test.go`
4. Consider impact on CPU instruction execution
5. Verify example programs still work correctly

## Error Handling

- Use custom error types defined in `err.go` files per package
- Wrap errors with context using `fmt.Errorf` with `%w`
- Return errors rather than panicking except for truly unrecoverable situations
- Log errors appropriately using the `translate` package for user-facing messages

## Internationalization

- Use the `translate` package for all user-facing strings
- Keep translation catalogs updated in `translate/locales/`
- Error messages should be translatable
- Follow golang.org/x/text patterns for i18n

## Performance Considerations

- CAPP operations are inherently parallel; consider this in implementations
- Track power/tick costs (constants defined in `emulator/emulator.go`)
- Avoid unnecessary cell iterations; use tagging efficiently
- Profile before optimizing; correctness and clarity first

## Code Review Focus Areas

1. **Documentation**: All exported identifiers properly documented?
2. **Testing**: Adequate test coverage for new code?
3. **Error Handling**: Errors properly propagated and handled?
4. **CAPP Semantics**: Operations correctly maintain CAPP invariants?
5. **Assembly Syntax**: Changes consistent with assembly language design?
6. **Backward Compatibility**: Existing programs and file formats still work?

## Resources

- Main README: `/README.md`
- CPU Documentation: `/cpu/README.md`, `/cpu/ASSEMBLY.md`, `/cpu/OPCODES.md`
- CAPP Documentation: `/capp/README.md`
- I/O Documentation: `/io/README.md`
- CLI Documentation: `/cmd/ucapp/README.md`
- Example Programs: `/examples/`
