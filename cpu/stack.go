package cpu

const (
	STACK_LIMIT = 16 // Maximum stack depth
)

type Stack struct {
	Data []uint32
}

func (s *Stack) Push(value uint32) {
	s.Data = append(s.Data, value)
}

func (s *Stack) Pop() (value uint32, ok bool) {
	value, ok = s.Peek()
	if ok {
		s.Data = s.Data[:len(s.Data)-1]
	}
	return
}

func (s *Stack) Empty() bool {
	return len(s.Data) == 0
}

func (s *Stack) Full() bool {
	return len(s.Data) == STACK_LIMIT
}

func (s *Stack) Peek() (value uint32, ok bool) {
	if s.Empty() {
		return
	}

	return s.Data[len(s.Data)-1], true
}

func (s *Stack) Reset() {
	if len(s.Data) > 0 {
		s.Data = s.Data[:0]
	}
}
