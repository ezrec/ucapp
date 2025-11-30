; Read from drum 1, ring 2, and erase bit 5, and write back to ring 3.
; Converts lowercase to UPPERCASE

.equ DEPOT_OP_SELECT $(0 << 23)
.equ DEPOT_OP_DRUM $(1 << 23)
.equ DRUM_OP_SELECT $(0 << 8)
.equ DRUM_OP_RING $(1 << 8)
.equ RING_OP_REWIND_READ 0
.equ RING_OP_REWIND_WRITE 1

.macro DEPOT_IN8 drum ring
; Select drum
alert depot $(DEPOT_OP_SELECT | drum)
await depot r0
; Select ring
alert depot $(DEPOT_OP_DRUM | DRUM_OP_SELECT | ring)
await depot r0
; Reset read pointer of ring
alert depot $(DEPOT_OP_DRUM | DRUM_OP_RING | RING_OP_REWIND_READ)
await depot r0
list of ARENA_FREE ARENA_MASK
list all
fetch depot 0xff
list not
list write ARENA_IO ARENA_MASK
.endm

.macro DEPOT_OUT8 drum ring
; Select drum
alert depot $(DEPOT_OP_SELECT | drum)
await depot r0
; Select ring
alert depot $(DEPOT_OP_DRUM | DRUM_OP_SELECT | ring)
await depot r0
; Reset write pointer of ring
alert depot $(DEPOT_OP_DRUM | DRUM_OP_RING | RING_OP_REWIND_WRITE)
await depot r0
store depot 0xff
.endm

DEPOT_IN8 1 2
list write 0x00 0x20
DEPOT_OUT8 1 3
