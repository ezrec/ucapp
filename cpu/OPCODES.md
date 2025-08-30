# Memory Layout

## Code Arena

### Compiled Instruction Area

```
   10 bit IP |          20 bit opcode |
             v                        v
10aa aaaa aaaa oooo oooo oooo oooo oooo
 ^
 | Arena ID (10) for code
```

Starting a program loads the CAPP into the CPU's IRAM; if the
CAPP is modified later it does _not_ affect the running program.

### Uncompiled Program Text

```
                      8 bit text item |
                                      v
1011 1111 1111 nnnn nnnn nnnn dddd dddd
               ^
               | 12 bit text index
```

# Instruction Encoding

## Value / Mask Sources

### RW registers

  0000 - r0
  0001 - r1
  0010 - r2
  0011 - r3
  0100 - r4
  0101 - r5
  0110 - ip
  0111 - stack

### RO registers

  1000 - match
  1001 - mask
  1010 - first
  1011 - count

  1100 - const 0
  1101 - const 0xffffffff
  1110 - immediate & 0xffff; immediate >> 16
  1111 - immediate; immediate >>= 32

## Channels

  000 - temp
  001 - depot
  010 - tape
  011 - VT
  100 - UNUSED
  101 - UNUSED
  110 - UNUSED
  111 - boot rom

## Channel Ops

  000 - fetch DMA to CAPP TAG items
  001 - store DMA from CAPP TAG items
  010 - await an alert.
  011 - alert the channel with a value/mask pair.

## ALU Arithmetic Ops

  000 - set
  001 - xor
  010 - and
  011 - or
  100 - shl
  101 - shr
  110 - add
  111 - sub

## Conditional (if) Ops

if some?
if none?
if true? A
if false? A
if eq? A B
if ne? A B
if lt? A B
if gt? A B
if le? A B
if ge? A B

    00 - == equals
    01 - != not equal
    10 - <  less than
    11 - <= less or equal

## CAPP Ops

  000 - swap
  001 - all
  010 - not
  011 - next
  100 - only VALUE MASK
  101 - of VALUE MASK
  110 - first VALUE MASK
  111 - write VALUE MASK

## Conditional prefixes

  00 - always execute
  01 - if cond is true
  10 - if cond is false
  11 - never execute

## ENCODING: cciixxooorrrvvvvmmmm

cc: Condition code
ii: Immediate subcode, or 0b00 for instruction
xx: Instruction class
aaa: ALU Operation
iii: If Operation
lll: CAPP list operation
rrr: Register ID/Channel number
vvvv: Value IR
mmmm: Mask IR

```
.alu.OP R V           16 - cc 000 aaa 0rrr VVVV
.if.OP A B            16 - cc 001 0ii AAAA BBBB
.list.OP V M          16 - cc 010 lll VVVV MMMM
.io OP CHANNEL ARG    16 - cc 011 0oo 0CCC AAAA
.imm YYYY             16 - YY YYY YYY YYYY YYYY -> imm = ((imm << 16) | 0xYYYY)
```

fetch CHANNEL MASK
store CHANNEL MASK
alert CHANNEL V
await CHANNEL V   - V must be a r0, r1, r2, r3, true?, false?, ip, stack

if some?
? jump foo
! jump bar

trap =>
 notify debug
 await  debug

### Execute an instruction

#### From registers

 - If IP has bit 31 set:
   - Read register IP & 7 as opcode
   - exit if IP >= (1 << 31) | 5

#### From CAPP

 - save CAPP match/mask
 - set CAPP match/mask to `ARENA_PROGRAM | IP` : `ARENA_MASK | IP_MASK`
 - exit if count != 1
 - opcode = first
 - restore CAPP match/mask
 - execute opcode
