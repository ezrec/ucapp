package capp

type Action int

//go:generate go tool stringer -type=Action
const (
	SET_SWAP  = Action(0) // Swap to alternate set
	LIST_ALL  = Action(1) // Enable tag bits if selected and matching.
	LIST_NOT  = Action(2) // Complement tags that are selected.
	LIST_NEXT = Action(3) // Disable first tagged entry

	LIST_ONLY   = Action(4) // MATCH/MASK: Disable non-matching tag bits
	SET_OF      = Action(5) // MATCH/MASK: Enable select bits if matching.
	WRITE_FIRST = Action(6) // VALUE/MASK: Write to first tagged entry
	WRITE_LIST  = Action(7) // VALUE/MASK: Modify all tagged cells
)
