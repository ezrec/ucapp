package cpu

const (
	STACK_LIMIT = 16 // Maximum stack depth
)

// Stack simulation state.
type Stack struct {
	Data []uint32
}

// Push a value on the stack.
func (s *Stack) Push(value uint32) {
	s.Data = append(s.Data, value)
}

// Pop a value from the stack.
func (s *Stack) Pop() (value uint32, ok bool) {
	value, ok = s.Peek()
	if ok {
		s.Data = s.Data[:len(s.Data)-1]
	}
	return
}

// Empty returns true if the stack if empty.
func (s *Stack) Empty() bool {
	return len(s.Data) == 0
}

// Full returns true if the stack is full.
func (s *Stack) Full() bool {
	return len(s.Data) >= STACK_LIMIT
}

// Peek at the current stack top.
// Returns ok=false if the stack is empty.
func (s *Stack) Peek() (value uint32, ok bool) {
	if s.Empty() {
		return
	}

	return s.Data[len(s.Data)-1], true
}

// Reset the stack.
func (s *Stack) Reset() {
	if len(s.Data) > 0 {
		s.Data = s.Data[:0]
	}
}
