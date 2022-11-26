package cache

import (
	"sync"
	"time"
)

var index = syncMap[string, time.Duration]{map_: &sync.Map{}}

type syncMap[keyType comparable, valueType any] struct {
	map_ *sync.Map
}

func (s *syncMap[keyType, _]) contains(k keyType) bool {
	_, ok := s.map_.Load(k)
	return ok
}

func (s *syncMap[keyType, valueType]) store(k keyType, v valueType) {
	s.map_.Store(k, v)
}

func (s *syncMap[keyType, _]) remove(k keyType) {
	s.map_.Delete(k)
}
