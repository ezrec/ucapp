package cpu

const (
	CAPP_SIZE = 1024
)

const (
	ARENA_MASK = 0xc_000_0000
	ARENA_IO   = 0x0_000_0000 // Input/Output.
	ARENA_TMP  = 0x4_000_0000 // Temporary.
	ARENA_CODE = 0x8_000_0000 // User code.
	ARENA_FREE = 0xc_000_0000 // Unused memory.
)
