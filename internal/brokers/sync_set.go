package brokers

import (
	"sync"

	"github.com/google/uuid"
)

type syncSet struct {
	set map[uuid.UUID]struct{}
	mtx sync.RWMutex
}

func (s *syncSet) add(id uuid.UUID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.set[id] = struct{}{}
}

func (s *syncSet) remove(id uuid.UUID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	delete(s.set, id)
}

func (s *syncSet) contains(id uuid.UUID) bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	_, ok := s.set[id]
	return ok
}
