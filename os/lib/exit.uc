; Return to shell
.macro DECLARE_OsLibExit
OsLibExit:
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
.endm
