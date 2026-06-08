; LIST the contents of the current drum
.include os/lib/exit.uc
.include os/lib/convert.uc

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
+ call OsLibConvert6To8
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

DECLARE_OsLibExit
DECLARE_OsLibConvert6to8
DECLARE_OsLibConvertData

