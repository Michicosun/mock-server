package util

import (
	"sync"
)

type SyncMap[K comparable, V any] struct {
	buffer map[K]V
	mtx    sync.RWMutex
}

func (mp *SyncMap[K, V]) Add(key K, value V) {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()
	mp.buffer[key] = value
}

func (mp *SyncMap[K, V]) Get(key K) (V, bool) {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	value, ok := mp.buffer[key]
	return value, ok
}

func (mp *SyncMap[K, V]) Remove(key K) {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()
	delete(mp.buffer, key)
}

func (mp *SyncMap[K, V]) Contains(key K) bool {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	_, ok := mp.buffer[key]
	return ok
}

func NewSyncMap[K comparable, V any]() SyncMap[K, V] {
	return SyncMap[K, V]{
		buffer: make(map[K]V),
	}
}
