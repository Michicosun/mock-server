package util

import (
	"sync"
)

type SyncSet[T comparable] struct {
	set map[T]struct{}
	mtx sync.RWMutex
}

func (s *SyncSet[T]) Add(key T) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.set[key] = struct{}{}
}

func (s *SyncSet[T]) Remove(key T) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	delete(s.set, key)
}

func (s *SyncSet[T]) Contains(key T) bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	_, ok := s.set[key]
	return ok
}

func NewSyncSet[T comparable]() SyncSet[T] {
	return SyncSet[T]{
		set: make(map[T]struct{}),
	}
}
