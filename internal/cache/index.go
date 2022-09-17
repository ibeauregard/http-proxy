package cache

import "sync"

var index = set{map_: &sync.Map{}}

type set struct {
	map_ *sync.Map
}

func (s *set) contains(e any) bool {
	_, ok := s.map_.Load(e)
	return ok
}

func (s *set) add(e any) {
	s.map_.Store(e, struct{}{})
}

func (s *set) remove(e any) {
	s.map_.Delete(e)
}
