# μCAPP Assembler Handbook

## Introduction

The μCAPP Assmbly language is a classic macro-assembler style language,
where there are:

- Equates (simple string replacements, ie `.equ NAME 2134`)
- Macros (complex multiline macros, with argments, ie `.macro ADD OUT A B`)
- Instruction primitives (such as `fetch`, `list all`, `jump LABEL`)

## System Equates

The following equates are predefined by the system:

| Name | Value |Comment |
| --- | --- | --- |
| `LINENO` | 1..N | Current source line number |
| `ARENA_MASK` | 0xc0000000 | Mask of the 'arena bits' in the CAPP word |
| `ARENA_IO`   | 0x00000000 | Arena ID for program IO usage. |
| `ARENA_DATA` | 0x40000000 | Arena ID for program data. |
| `ARENA_CODE` | 0x80000000 | Arena ID for program code. |
| `ARENA_FREE` | 0xc0000000 | Arena ID for unused CAPP words. |
| `CAPP_SIZE`  | 8192 | The total size of the CAPP, in words. |

## Instruction Primitives

The assembler accepts instructions in the following format:

`[CONDITION] CATEGORY ACTION [ARG1 [ARG2 [ARG3]]]`

CONDITION, if present, is one of '?' or '!'.

CATEGORY is one of 'list', 'alu', 'if', or 'io'.

See the following subchapters for the ACTIONs for each category.

### Conditional Execution

CONDITION may be one of the following:

| CONDITION | Comment |
| --- | --- |
| `?` | Execute if and only if CPU COND bit is true. |
| `!` | Execute if and only if CPU COND bit is false. |

If CONDITION is not specifed, the instruction is always executed.

### Registers

#### Read/Write

| Name | Comment |
| --- | --- |
| `r0`..`r5` | General purpose registers 0..5 |
| `ip` | Instruction pointer |
| `stack` | On read, pop from top of stash. On write, push to top of stack. |

#### Read Only

| Name | Comment |
| --- | --- |
| `match` | Last value from `list of`'s MATCH component. |
| `mask` | Last value from `list of`'s MASK component. |
| `first` | Value of the first tagged item in the active CAPP list, or zero if none. |
| `count` | Number of tagged items in the active CAPP list. |

## CAPP Instructions

Each CAPP cell has a 32-bit data area, a SELECTED bit, and a TAGGED bit.

The CAPP can be directly accessed by the `list` family of instructions,
as follows.

### list of MATCH [MASK]

Operates over: All CAPP cells.

CAPP cells which contain bits in MASK that have the value of the corresponding bits in MATCH will be marked as SELECTED.
All other cells will have their SELECTED bit cleared.

The TAGGED bit is left unchanged by this operation.

| Argument | Class |
| ---      | --- |
| MATCH    | Immediate or Register |
| MASK     | Immediate or Register |

NOTE: If MASK not specified, MASK defaults to 0xFFFFFFFF,
      and only cells that exactly match MATCH will be selected.

### list all

Operates over: SELECTED cells.

All CAPP items currently SELECTED will have their TAGGED bit set.

Takes no arguments.

### list not

Operates over: SELECTED cells.

Inverts the TAGGED state of all currently SELECTED CAPP items.
TAGGED items will be untagged, and untagged items will become TAGGED.

Takes no arguments.

### list only MATCH [MASK]

Operates over: SELECTED and TAGGED cells.

Untag SELECTED and TAGGED cells whose bits in MASK do not match
the corresponding bits in MATCH.

| Argument | Class |
| --- | --- |
| MATCH | Immediate or Register |
| MASK | Immediate or Register |

NOTE: If MASK not specified, MASK defaults to 0xFFFFFFFF,
      and only value that exactly match MATCH will be left in the list.

### list write VALUE [MASK]

Operates over: SELECTED and TAGGED cells.

Update all SELECTED and TAGGED CAPP cells, changing the
bits specified in MASK to the corresponding bit values in VALUE.

NOTE: If MASK not specified, MASK defaults to 0xFFFFFFFF,
      and all SELECTED and TAGGED items' value will be replaced with VALUE.

NOTE: The alternate syntax for this operation is `write list VALUE [MASK]`

### list first VALUE [MASK]

Operates over: First SELECTED and TAGGED cell.

Update the first SELECTED and TAGGED CAPP cell, changing the
bits specified in MASK to the corresponding bit values in VALUE.

NOTE: If MASK not specified, MASK defaults to 0xFFFFFFFF,
      and first item in the current list will be replaced with VALUE.

NOTE: The alternate syntax for this operation is `write first VALUE [MASK]`

### list next

Operates over: First SELECTED and TAGGED cell.

Untag the first item in the list, if `count` > 0.

Takes no arguments.

## ALU Instructions

All ALU instructions are performed serially on the CPU, and have the
following format:

`alu OP TARGET VALUE [MASK]`

| Argument | Comment |
| --- | --- |
| OP | ALU Operation |
| TARGET | Writable Register |
| VALUE | Immediate or Register |
| MASK  | Immediate or Register |

ALU operations modify the target by the specified operation, using the VALUE anded with the MASK as the operand.

`TARGET := TARGET OP (VALUE & MASK)`

NOTE: If MASK is not specified, it is assumed to be 0xFFFFFFFF,
      leaving VALUE unmodified.

### ALU Operations

#### alu set TARGET VALUE MASK

Set TARGET to (VALUE & MASK).

NOTE: The alternate syntax for this operation is `write TARGET VALUE [MASK]`

#### alu xor TARGET VALUE MASK

Set TARGET to TARGET XOR (VALUE & MASK)

#### alu and TARGET VALUE MASK

Set TARGET to TARGET AND (VALUE & MASK)

#### alu or TARGET VALUE MASK

Set TARGET to TARGET OR (VALUE & MASK)

#### alu shl TARGET VALUE MASK

Set TARGET to TARGET << (VALUE & MASK).

#### alu shr TARGET VALUE MASK

Set TARGET to TARGET >> (VALUE & MASK).

#### alu add TARGET VALUE MASK

Set TARGET to TARGET + (VALUE & MASK).

#### alu sub TARGET VALUE MASK

Set TARGET to TARGET - (VALUE & MASK).

## Conditional Instructions

The conditional instructions allow setting the COND single bit register
of the CPU, used for conditional execution of instructions.

`if COND SRCA SRCB`

| Argument | Comment |
| --- | --- |
| COND | Conditional Operation |
| SRCA | Immediate or Register |
| SRCB | Immediate or Register |

NOTE: For all operations, SRCA and SRCB are treated as signed 32-bit integers.

### if eq? SRCA SRCB

Set COND to true if SRCA == SRCB, false otherwise.

### if ne? SRCA SRCB

Set COND to true if SRCA != SRCB, false otherwise.

### if lt? SRCA SRCB

Set COND to true if SRCA < SRCB, false otherwise.

### if le? SRCA SRCB

Set COND to true if SRCA <= SRCB, false otherwise.

### if gt? SRCA SRCB

Set COND to true if SRCA > SRCB, false otherwise.

### if ge? SRCA SRCB

Set COND to true if SRCA >= SRCB, false otherwise.

### if some?

Set COND to true if `count` > 0, false otherwise.

NOTE: This is the alternate syntax for `if gt? count 1`

### if none?

Set COND to true if `count` == 0, false otherwise.

NOTE: This is the alternate syntax for `if eq? count 0`

### if true? SRCA

Set COND to true if `count` == 0, false otherwise.

NOTE: This is the alternate syntax for `if eq? SRCA 0`

## I/O Instructions

The io instructions provide an interface with off-μCAPP devices, by
sending CAPP data to the device, receiving CAPP data from the device,
alerting the device with control words, or awaiting a response
from the device.

`io OP CHANNEL ARG1 ARG2`

| Argument | Comment |
| --- | --- |
| OP | I/O Operation. |
| CHANNEL | Channel ID number. |
| ARG1 | Immediate or Register. |
| ARG2 | Immediate or Register. |

### I/O Channels

| Name | ID | Bit Width | Comment |
| --- | --- | --- | --- |
| temp | 0 | 1 | Temporary IO channel. |
| depot | 1 | 1 | Drum and Ring depot. |
| tape | 2 | 1 | Linear input/output tape. |
| vt | 3 | 24 | Virtual Terminal. |
| monitor | 4 | 32 | Boot ROM and debug monitor. |

For more details on channel specific IO features, see [channel/README.md](channel/README.md)

### I/O Operations

#### io fetch CHANNEL VALUE MASK

Fetch the channel bitstream, and write to tagged items in the CAPP.

The MASK argument defines the locations in the CAPP cell that will be
replaced by the bits from the channel bitstream, in LSB to MSB order.
The CAPP's cell content is ORed with VALUE after the replacement.

For example, if the channel bitsream contained 1,0,1,0,1,1,0,0,...;
and MASK was 0x00000f0f, and the first tagged cell was 0, and VALUE
was `0x1234_0000` it would be replaced with
`0b0001_0010_0011_0100_0000_0011_0000_0101` (0x12340305).

If MASK is omitted, all 32 bits of each CAPP cell is fetched. If VALUE is
omitted, it is assumed to be zero.

Once all of the bits from BITS are loaded into the first SELECTED and
TAGGED CAPP cell, the cell's TAGGED bit will be set to false, and
IO will proceed with the next SELECTED and TAGGED CAPP cell.

If not enough CAPP cells are available to read the entire bitstream,
the next `io fetch CHANNEL` operation for this channel will continue
from where it left off.

#### io store CHANNEL [VALUE [MASK]]

Store the SELECTED and TAGGED cells of the CAPP to the channel
bitstream.

The MASK argument defines the locations in the CAPP cell that will be
stored to the channel bitstream, in LSB to MSB order. VALUE is ORed with
the cell's data before sending to the channel bitstream.

If MASK is omitted, all 32 bits of each CAPP cell is stored. If VALUE is
omitted, it is assumed to be zero.

Once all of the bits from MASK are stored from the first SELECTED and
TAGGED CAPP cell, the cell's TAGGED bit will be set to false, and
IO will proceed with the next SELECTED and TAGGED CAPP cell.

#### io alert CHANNEL VALUE [MASK]

Send an alert to the channel. See channel specific documentation for
various alert codes.

#### io await CHANNEL [REG]

Halt the CPU until data is available on the channel. If REG is
a valid writable register, then the awaited value is stored in REG.

### Flow Control

#### exit

Halt the CPU, and leave the emulation.

#### jump LABEL

Jump to a to a LABEL

#### vjump VALUE [MASK]

Jump to (VALUE & MASK) instead of a label.

#### call LABEL

Push the next IP to the stack, and jump to LABEL.

#### vcall VALUE [MASK]

Call to the IP at (VALUE & MASK), instead of a label.

#### return

Pop the stack into the IP register, returning from a call or vcall.
