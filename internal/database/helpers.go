package database

import "sync"

func runWithWriteLock(mutex *sync.RWMutex, task func() error) error {
	mutex.Lock()
	defer mutex.Unlock()
	return task()
}

func runWithReadLock[T any](mutex *sync.RWMutex, task func() (T, error)) (T, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return task()
}
