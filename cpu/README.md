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
  1101 - const 1
  1110 - const 0xffffffff
  1111 - immediate & 0xffffffff; immediate >>= 32

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
  - .if.eq 
if none?
if true? A
if false? A
if eq? A B
if ne? A B
if lt? A B
if gt? A B
if le? A B
if ge? A B

  1000 - ==  equals
  1001 - <  less than
  1010 - >  greater than
  1011 - ?

  1100 - !=  not equals
  1101 - !<  greater or equal to
  1110 - !>  less than or equal to
  1111 - ?

## CAPP Ops

  000 - idle
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
.alu.OP R V M         - cc00 00aa arrr VVVV MMMM
.if.OP R A B          - cc00 01ii i000 AAAA BBBB
.list.OP V M          - cc00 10ll l000 VVVV MMMM
.io OP CHANNEL V M    - cc00 11oo oCCC VVVV MMMM
.imm_lo32  YYYY       - cc01 + 16 bit data -> imm = imm << 32 | 0x0000YYYY
.imm_hi32  YYYY       - cc10 + 16 bit data -> imm = imm << 32 | 0xYYYY0000
.imm_or16  YYYY       - cc11 + 16 bit data -> imm = imm | 0x0000YYYY
```

alert CHANNEL V M
await CHANNEL V M   - V must be a r0, r1, r2, r3, true?, false?, ip, stack

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
