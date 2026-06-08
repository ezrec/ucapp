; Support functions
; Convert 6 bit name to 4x8 bit.
; r0 - lower 3 bytes contains the 6-bit name.
.macro DECLARE_OsLibConvert6to8
OsLibConvert6To8:
alu set r2 0
alu set r3 0
alu set stack mask
alu set stack match
list of $(ARENA_DATA | (0 << 14)) $(ARENA_MASK | (0x3 << 14))
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
.endm

.macro DECLARE_OsLibConvert8To6
OsLibConvert8To6:
; Convert 4x8 bit command in r0 to 6-bit encoding
alu set r2 0
alu set stack mask
alu set stack match
list of ARENA_DATA ARENA_MASK
CONVERT:
if eq? r0 0
- list all
- alu set r1 r0
- alu shr r1 24
- list only r1 0xff
- alu set r1 first
- alu shl r1 10
- alu and r1 0x00fc0000
- alu shr r2 6
- alu or r2 r1
- alu shl r0 8
- jump CONVERT
alu set r0 r2
list of stack stack
return
.endm

.macro DECLARE_OsLibConvertData
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
.dw $((16 << 8) | 0x40) ; '@'
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
.dw $((1 << 14) | (17 << 8) | 'a')
.dw $((1 << 14) | (18 << 8) | 'b')
.dw $((1 << 14) | (19 << 8) | 'c')
.dw $((1 << 14) | (20 << 8) | 'd')
.dw $((1 << 14) | (21 << 8) | 'e')
.dw $((1 << 14) | (22 << 8) | 'f')
.dw $((1 << 14) | (23 << 8) | 'g')
.dw $((1 << 14) | (24 << 8) | 'h')
.dw $((1 << 14) | (25 << 8) | 'i')
.dw $((1 << 14) | (26 << 8) | 'j')
.dw $((1 << 14) | (27 << 8) | 'k')
.dw $((1 << 14) | (28 << 8) | 'l')
.dw $((1 << 14) | (29 << 8) | 'm')
.dw $((1 << 14) | (30 << 8) | 'n')
.dw $((1 << 14) | (31 << 8) | 'o')
.dw $((1 << 14) | (32 << 8) | 'p')
.dw $((1 << 14) | (33 << 8) | 'q')
.dw $((1 << 14) | (34 << 8) | 'r')
.dw $((1 << 14) | (35 << 8) | 's')
.dw $((1 << 14) | (36 << 8) | 't')
.dw $((1 << 14) | (37 << 8) | 'u')
.dw $((1 << 14) | (38 << 8) | 'v')
.dw $((1 << 14) | (39 << 8) | 'w')
.dw $((1 << 14) | (40 << 8) | 'x')
.dw $((1 << 14) | (41 << 8) | 'y')
.dw $((1 << 14) | (42 << 8) | 'z')
.endm
