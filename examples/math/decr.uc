; Example program to increment the bytes on the input tape by 1, and output
; to the output tape. Uses as much CAPP is as available per input chunk,
; and can process an arbitrarily long input tape. 0xff + 1 => 0x00

More:
; Load tape as lower 8 bits of newly allocated CAPP
list of ARENA_FREE ARENA_MASK
list all
fetch tape 0xff
list not
write list ARENA_IO 0xffffff00
list of ARENA_IO ARENA_MASK
if none?
? exit

; Mark all with the TODO bit (1 << 8)
.equ TODO $(1 << 8)
write list TODO TODO

.equ RCOUNT r0
.equ RMASK r1
write RCOUNT 8
write RMASK 1
Loop:
list of $(ARENA_IO | TODO) $(ARENA_MASK | TODO)
list only RMASK RMASK ; Select ones, sub one and remove TODO bit
write r2 RMASK
alu or r2 TODO
write list 0 r2 ; Write 1 as 0, clear TODO
list not ; Select 0s
write list RMASK RMASK ; Write 0 as 1, leave TODO
alu shl RMASK 1
alu sub RCOUNT 1
if eq? RCOUNT 0
! jump Loop

list of ARENA_IO ARENA_MASK
list all
store tape 0xff ; Store only the low bytes of the tape.
list not
write list ~0 ~0
jump More
