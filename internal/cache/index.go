package cache

var index = set{map_: map[any]struct{}{}}

type set struct {
	map_ map[any]struct{}
}

func (s *set) contains(e any) bool {
	_, ok := s.map_[e]
	return ok
}

func (s *set) add(e any) {
	s.map_[e] = struct{}{}
}

func (s *set) remove(e any) {
	delete(s.map_, e)
}
