; LIST the contents of the current drum

; Load the contents of ring 0xff
alert depot $(DEPOT_OP_DRUM | DRUM_OP_SELECT | 0xff)
await depot
alert depot $(DEPOT_OP_DRUM | DRUM_OP_RING | RING_OP_REWIND_READ)
await depot
list of CAPP_FREE
list all
fetch depot
list not

; Remove deleted entries
list only 0xff000000 0xff000000
write list ~0
list all
list only ~0
list not

; Clear out ring numbers, and move to ARENA_IO
write list ARENA_IO 0xff000000
list of ARENA_IO ARENA_MASK
list all

; Print out ring names
PRINT_ONE:
if some?
+ alu set r0 first
+ call Convert6To8
+ write first r0
+ list next
+ list not
+ store tape r3
+ list not
+ write first 0x0a
+ list next
+ list not
+ store tape 0xff
+ list not
+ write first CAPP_FREE
+ list next
+ jump PRINT_ONE

; Return to shell
alert depot $(DEPOT_OP_DRUM | DRUM_OP_SELECT | 0x00)
await depot
alert depot $(DEPOT_OP_DRUM | DRUM_OP_RING | RING_OP_REWIND_READ)
await depot
; Load regs with boot program
alu set r0 0x15cc ; list of 0 0
alu set r1 0x11cc ; list all
alu set r2 0x17dd ; list write ~0
alu set r3 0x181d ; fetch depot
alu set r4 0x12cc ; list not
alu set r5 0x006c ; alu set ip 0

; Switch IP to boot-from-registers
alu set ip IP_MODE_REG

; Support functions
Convert6To8:
; Convert 6 bit name to 4x8 bit
alu set r2 0
alu set r3 0
alu set stack mask
alu set stack match
list of ARENA_DATA ARENA_MASK
CONVERT:
if eq? r0 0
- alu set r1 r0
- alu shr r1 10
- list all
- list only r1 0x3f00
- alu set r1 first
- alu and r1 0xff
- alu shl r2 8
- alu or r2 r1
- alu shl r0 6
- alu and r0 ~0x3f000000
- alu shl r3 8
- alu or r3 0xff
- jump CONVERT
alu set r0 r2
list of stack stack
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

