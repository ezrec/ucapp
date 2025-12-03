; SHELL - Drum 0, Ring 0 boot program
;
; Does a partial read of the Tape input channel to determine the
; command to run. Then loads the ring associated with the command,
; and executes it. The TEMPORARY channel will contain the command
; after the name of the command.

PROMPT:
; Print the shell prompt.
list of ARENA_FREE ARENA_MASK
list all
write first 0x617264 ; 'dra'
list next
write first 0x203e74 ; 't> '
list next
list not
store tape 0xffffff
list not
write list ~0

; Read command from command line
; (first 4 bytes or until a space is seen)
alu set r0 0
list of ARENA_FREE ARENA_MASK
list all
list next
list not    ; Only one word is allocated in the list.
write first ARENA_IO ARENA_MASK
list of ARENA_IO ARENA_MASK

NEXT_LETTER:
list all
write first 0
fetch tape 0xff
list not
if none?
+ jump NEXT_LETTER
if eq? first ' ' ; command complete?
- if eq? first '\n' ; command complete?
- alu shl r0 8
- write first r0 0xffffff00
- alu set r0 first
- jump NEXT_LETTER

; Write remainder of command line to TEMPORARY channel
write first 0 0xffffff00
NEXT_COMMAND:
if eq? first '\n' ; command complete?
- fetch tape 0xff
- list not
- if none?
- store temp 0xff
- list not
- jump NEXT_COMMAND

call Convert8To4

; Find command (in r0) in current drum's Ring 0xff directory
LOAD_RING:
alert depot $(DEPOT_OP_DRUM | DRUM_OP_SELECT | 0xff)
await depot
alert depot $(DEPOT_OP_DRUM | DRUM_OP_RING | RING_OP_REWIND_READ)
await depot
list of ARENA_FREE ARENA_MASK
list all
fetch depot
list not
alu shl r0 8
list only r0 0xffffff00
if none?
+ list all
+ list write ARENA_FREE ARENA_MASK
+ jump PROMPT

; Switch to ring for command
alu set r0 first
list write ARENA_FREE ARENA_MASK
alu and r0 0xff
alu or r0 $(DEPOT_OP_DRUM | DEPOT_OP_SELECT)
alert depot r0
await depot r0
if eq? r0 ~0
+ jump PROMPT

; Load regs with boot program
alu set r0 0x15cc ; list of ~0 ~0
alu set r1 0x11cc ; list all
alu set r2 0x17dd ; list write ~0
alu set r3 0x181d ; fetch depot
alu set r4 0x12cc ; list not
alu set r5 0x000c ; alu set ip 0

; Switch IP to boot-from-registers
alu set ip IP_MODE_REG

; Support functions
Convert8To4:
; Convert 4x8 bit command in r0 to 6-bit encoding
alu set r2 0
list of ARENA_DATA ARENA_MASK
CONVERT:
if eq? r0 0
- list all
- list only r0 0xff
- alu set r1 first
- alu shr r1 8
- alu and r1 0x3f
- alu shl r2 6
- alu or r2 r1
- alu shr r0 8
- jump CONVERT
alu set r0 r2
return

.dw $((0 << 8) | 0)
.dw $((1 << 8) | '1')
.dw $((2 << 8) | '2')
.dw $((3 << 8) | '3')
.dw $((4 << 8) | '4')
.dw $((5 << 8) | '5')
.dw $((6 << 8) | '6')
.dw $((7 << 8) | '7')
.dw $((8 << 8) | '8')
.dw $((9 << 8) | '9')
.dw $((10 << 8) | '0')
.dw $((11 << 8) | '+')
.dw $((12 << 8) | '-')
.dw $((13 << 8) | '_')
.dw $((14 << 8) | '.')
.dw $((15 << 8) | ',')
.dw $((16 << 8) | '@')
.dw $((17 << 8) | 'A')
.dw $((18 << 8) | 'B')
.dw $((19 << 8) | 'C')
.dw $((20 << 8) | 'D')
.dw $((21 << 8) | 'E')
.dw $((22 << 8) | 'F')
.dw $((23 << 8) | 'G')
.dw $((24 << 8) | 'H')
.dw $((25 << 8) | 'I')
.dw $((26 << 8) | 'J')
.dw $((27 << 8) | 'K')
.dw $((28 << 8) | 'L')
.dw $((29 << 8) | 'M')
.dw $((30 << 8) | 'N')
.dw $((31 << 8) | 'O')
.dw $((32 << 8) | 'P')
.dw $((33 << 8) | 'Q')
.dw $((34 << 8) | 'R')
.dw $((35 << 8) | 'S')
.dw $((36 << 8) | 'T')
.dw $((37 << 8) | 'U')
.dw $((38 << 8) | 'V')
.dw $((39 << 8) | 'W')
.dw $((40 << 8) | 'X')
.dw $((41 << 8) | 'Y')
.dw $((42 << 8) | 'Z')

