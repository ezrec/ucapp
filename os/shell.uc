; SHELL - Drum 0, Ring 0 boot program
;
; Does a partial read of the Tape input channel to determine the
; command to run. Then loads the ring associated with the command,
; and executes it. The TEMPORARY channel will contain the command
; after the name of the command.

.include os/lib/exit.uc
.include os/lib/convert.uc

PROMPT:
; Dump temporary
list of CAPP_FREE
list all
fetch temp
list not
write list CAPP_FREE
; Print the shell prompt.
list of CAPP_FREE
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
list of CAPP_FREE
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
+ exit  ; FIXME: how to best exit?
+ jump NEXT_LETTER
if eq? first ' ' ; command complete?
- if eq? first '\n' ; command complete?
- if eq? first '\r' ; command complete?
- alu shl r0 8
- write first r0 0xffffff00
- alu set r0 first
- jump NEXT_LETTER

; Write remainder of command line to TEMPORARY channel
write first 0 0xffffff00
NEXT_COMMAND:
if eq? first '\n' ; command complete?
- if eq? first '\r' ; command complete?
- fetch tape 0xff
- list not
- if none?
- store temp 0xff
- list not
- jump NEXT_COMMAND

call OsLibConvert8To6

; Find command (in r0) in current drum's Ring 0xff directory
LOAD_RING:
alert depot $(DEPOT_OP_DRUM | DRUM_OP_SELECT | 0xff)
await depot
alert depot $(DEPOT_OP_DRUM | DRUM_OP_RING | RING_OP_REWIND_READ)
await depot
list of CAPP_FREE
list all
fetch depot
list not
list only r0 0x00ffffff
if none?
+ list all
+ list write CAPP_FREE
+ jump PROMPT

; Switch to ring for command
alu set r0 first
list write CAPP_FREE
alu shr r0 24
alu or r0 $(DEPOT_OP_DRUM | DEPOT_OP_SELECT)
alert depot r0
await depot r0
if eq? r0 ~0
+ jump PROMPT

DECLARE_OsLibExit
DECLARE_OsLibConvert8To6
DECLARE_OsLibConvertData
