# I/O Peripherals

The μCAPP can access a number of I/O peripherals, which are mapped as follows:

| Channel | Type | Purpose |
| ---     | ---  | ---     |
| 0       | Temp | Temporary store |
| 1       | Depot | Persistent storage |
| 2       | Tape | Input/Output linear tape |
| 3       | VT   | Virtual Terminal   |
| 4       | n/a | unused |
| 5       | n/a | unused |
| 6       | debug | Debug channel |
| 7       | rom | OS ROM image |

## Temp

### IO Operations

```
store temp MASK ; store bits from tagged cells in MASK to temp
fetch temp MASK ; fetch bit data from temp into tagged cells according to MASK
```

## Depot

A depot is a storage device, which consists of several storage drums (each with a unique ID), and each storage drum has up to 256 rings, which can each store up to 64K bytes of data.

### IO Operations

```
fetch depot MASK ; fetch currently selected drum/ring into bits of tags in MASK
store depot MASK ; append tags' MASK bits to currently selected drum/ring
alert depot ((0x0 << 23) | DRUM) ; Select a specific drum (20 bits)
alert depot ((0x1 << 23) | (0 << 8) | RING) ; Select a specific ring (8 bits).
alert depot ((0x1 << 23) | (1 << 8) | 0) ; Reset current ring's read pointer.
alert depot ((0x1 << 23) | (1 << 8) | 1) ; Reset current ring's write pointer.
```

NOTE: A read of a ring cannot go past its current write pointer.

### Drum

A drum is a storage device for multiple rings of data, which can be read or written to, used as a program library, or any number of other purposes.

A drum is divided into rings, each of which can store up to 64K of byte oriented data.

### Rings

A `ring` is simply an array of 8-bit data.

#### Compiling a ucapp program to a ring.

When `ucapp` is run with the `--dump <file.ring>` option, a dump of the CAPP before the program would have been executed will be saved to the `<file.ring>` parameter.

```
$ go run ./cmd/ucapp --dump hello_world.ring example/hello_world.ucapp
$ go run ./cmd/ucapp hello_world.ring
Hello World!
```

### Ring File Format

- 4-byte magic word `\xb5RNG`
- 1-byte length of metadata including this byte (set to `8` in this version)
- 1-byte length (as a power of 2, up to 32) of the ring.
- 1-byte unused (set at zero)
- 1-byte unused (set at zero)
- 4-byte (little endian) last-written-value index.
- The remainder is a byte array.

## Drums

### Drum File Format

- 4-byte magic word `\xb5DRM`
- 1-byte length of metadata including this byte (set to `4` in this version)
- 1-byte length (as a power of 2, up to 32) of the CAPP.
- 1-byte (unused, set as zero)
- 1-byte (unused, set as zero)
- For each ring:
    - A `\xb5RNG` formatted ring.


### Executing a Drum

To execute a drum, use the `--drum` option:

```
$ go run ./cmd/ucapp --verbose --drum example.drum
ucapp: Drum has 3 rings, executing last ring.
This is ring 3! Calling ring 2...
This is ring 2! Calling ring 1...
This is ring 1! Hello World!
.. back in ring 2.
.. back in ring 3.
ucapp: Drum exited.
```

## Tape

A tape is a byte stream, and can be specified to the μCAPP emulator with the `--tape-input` and `--tape-output` parameters (default are stdin and stdout, respectively). The input tape can only be sequentially read, and the output tape any only be sequentially written to.

### IO Operations

```
fetch tape MASK ; Load input tape into MASK bits of tagged cells.
store tape MASK ; Append MASK bits of tagged cells to output tape.
```

### Reading

If insufficient tagged CAPP entries are available for the read, the remainder of the tape segment is left unread.

## Virtual Terminal (VT)

The Virtual Terminal (VT) provides a full screen UTF-8 terminal.

### IO Operations

```
fetch vt 0xFF ; Read 8-bit keycode from the VT input buffer to tagged cells
store vt 0x3FFF_FFFF      ; Write tagged cells to VT display
```

### Reading

Using the `fetch vt 0xff` opcode, the tagged CAPP cells' lower 8 bits are replaced with the keystroke scan code of the next keys in the VT key queue.

The VT can address a matrix of up to 128x64 cells.

See [io/KEYMAP.md](io/KEYMAP.md) for the complete key mapping.

### Writing

Using the `store vt 0x3FFF_FFFF`, modify the VT's frame buffer with the tagged CAPP cells' lower 24 bits.

| Bits | Purpose | Comment |
| ---- | ---     | --- |
| 0..7 | Glyph | Glyph value |
| 8 | Unused | Unused |
| 9 | Unused | Unused |
| 10 | 0 | Indicates cell content write |
| 11..16 | Row    | Row number 0..63 |
| 17..23 | Column | Column number 0..127 |

| Bits | Purpose | Comment |
| ---- | ---     | --- |
| 0..3 | Foreground | Cell foreground color |
| 4..7 | Background | Cell background color |
| 8 | Bold | Set to make the cell bold |
| 9 | Italic | Set to make the cell italic |
| 10 | 1 | Indicates cell attribute write |
| 11..16 | Row    | Row number 0..63 |
| 17..23 | Column | Column number 0..127 |

The VT will always use the most recently written row/col value.

## Monitor

The Monitor channel contains the boot ROM for the CPU, and is the target for inter-drum communication. It is bitstream of 32 bit wide words (2 bits of arena ID, 10 bits of IP data, and 20 bits of opcode), which is loaded in at machine reset.

See [cpu/README.md](cpu/README.md) for bootstrap details.

After bootstrap is complete, the Monitor is available for inter-process communication use between drums.

### Inter-Process Communication

Inter-process communication can be done between any two drums in the depot.

#### Call Procedure

```
; select a drum to communicate with. Ring should be set to 0xff (active CPU)
io alert depot $((DRUM_ID << 8) | 0xff)
; await the drum's readyness.
io await depot r0
; (optional) check return code
if ne? r0 0
? exit
; send the IPC key for the function call to the Monitor
io alert monitor SOME_IPC_FUNCTION_KEY
; (optional) send any parameter data
io store monitor
list not
list write ARENA_FREE ARENA_MASK
; (optional) fetch any result data
list of ARENA_FREE ARENA_MASK
list all
io fetch monitor
list not
; await response code
io await monitor r0
; (optional) check return code
if ne? r0 0
? exit
```

#### Inter-Process Communication Server

```
LOOP:
; await a remote IPC on the monitor
io await monitor r0
; (optional) read data from the monitor channel
list of ARENA_FREE ARENA_MASK
list all
io fetch monitor
list not
; switch based off the key to the action to perform
if eq? r0 SOME_IPC_FUNCTION_KEY
? call SOME_IPC_FUNCTION_HANDLER ; result is in r0
; (optional) store result data to the monitor channel
io store monitor
list not
; alert with result code
io alert monitor r0
```
