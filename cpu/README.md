# μCAPP CPU

## Assembly Language

- [ASSEMBLY.md](μCAPP Assembler Handbook).

## CPU Opcodes

- [OPCODES.md](μCAPP CPU Opcode Layout)

## Reset

### Initial CPU state

The reset state of the machine is:

- All CAPP values are randomized.
- CPU registers are preloaded with the following instructions:
  - r0: `.list.of 0 immz immz         ; Select all of the CAPP`
  - r1: `.list.all 0 immz immz        ; Tag all items`
  - r2: `.list.write 0 immnz immnz    ; Replace all values with 0xFFFFFFFF`
  - r3: `.io.fetch rom immz immnz     ; Load boot ROM into CAPP`
  - r4: `.list.not 0 immz immz        ; Now, only the program is tagged`
  - r5: `.alu.set ip immz immnz      ; Set IP to 0x00000000 (exec from CAPP)`
- CPU IP is set to 0x8000000 (execute-from-registers)

### Bootstrap

- CPU executes code in registers
- CPU's IP is set to 0x00000000 by code in r5
- OS boot code is as follows:
  - Select Drum 0, Ring 255 from the Depot
  - Read into CAPP as a program in IO arena.
  - Select boot program in CAPP
  - Write trampoline into registers:
    - r0: `.list.write 0 immnz immnz  ; Free boot program`
    - r1: `.imm_hi32 0x8000           ; Arena ID for program`
    - r2: `.imm_hi32 0xc000           ; Arena mask`
    - r3: `.list.of 0 immz imm        ; Select IO program in IO area`
    - r4: `.list.write 0 immnz imm    ; Write program ID to list`
    - r5: `.alu.set ip immz immnz    ; Set IP to 0x00000000 (exec from CAPP)`
  - CPU IP is set to 0x8000000 (execute-from-registers)
  - Control is transferred to the program from Drum 0, Ring 255.
