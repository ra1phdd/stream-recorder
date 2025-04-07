package m3u8

type OrderedSet struct {
	order []string
	set   map[string]struct{}
}

func NewOrderedSet() *OrderedSet {
	return &OrderedSet{
		order: make([]string, 0),
		set:   make(map[string]struct{}),
	}
}

func (s *OrderedSet) Get() []string {
	return s.order
}

func (s *OrderedSet) Add(value string) {
	if _, exists := s.set[value]; !exists {
		s.set[value] = struct{}{}
		s.order = append(s.order, value)
	}
}

func (s *OrderedSet) Has(value string) bool {
	_, exists := s.set[value]
	return exists
}

func (s *OrderedSet) Delete(value string) {
	if _, exists := s.set[value]; !exists {
		return
	}
	delete(s.set, value)

	for i, v := range s.order {
		if v == value {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
}

func (s *OrderedSet) Len() int {
	return len(s.order)
}

func (s *OrderedSet) TrimToLast(n int) {
	if n >= len(s.order) {
		return
	}

	for _, v := range s.order[:len(s.order)-n] {
		delete(s.set, v)
	}

	s.order = s.order[len(s.order)-n:]
}

func (s *OrderedSet) Clear() {
	s.set = make(map[string]struct{})
	s.order = s.order[:0]
}
