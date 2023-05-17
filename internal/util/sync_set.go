package util

import (
	"sync"
)

type SyncSet[K comparable] struct {
	buffer map[K]struct{}
	mtx    sync.Mutex
}

func (s *SyncSet[K]) Insert(value K) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, ok := s.buffer[value]

	if ok {
		return false
	}

	s.buffer[value] = struct{}{}
	return true
}

func (s *SyncSet[K]) Remove(value K) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	delete(s.buffer, value)
}

func NewSyncSet[K comparable]() SyncSet[K] {
	return SyncSet[K]{
		buffer: make(map[K]struct{}),
	}
}
