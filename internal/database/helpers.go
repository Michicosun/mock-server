package database

import "sync"

func runWithLock[T any](mutex *sync.Mutex, task func() T) T {
	mutex.Lock()
	defer mutex.Unlock()
	return task()
}
