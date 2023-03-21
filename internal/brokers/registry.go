package brokers

import (
	"fmt"
	"mock-server/internal/util"
	"sync"
)

var MPRegistry = &mpRegistry{}

type MessagePoolHandler interface {
	NewReadTask() qReadTask
	NewWriteTask(data [][]byte) qWriteTask
}

type MessagePool interface {
	getName() string
	getBroker() string
	getHandler() MessagePoolHandler
}

type mpRegistry struct {
	constructor sync.Once
	registry    util.SyncMap[string, MessagePool]
}

func (r *mpRegistry) Init() {
	r.constructor.Do(func() {
		r.registry = util.NewSyncMap[string, MessagePool]()
		// fetch db
	})
}

func (r *mpRegistry) AddMessagePool(pool MessagePool) (MessagePoolHandler, error) {
	if r.registry.Contains(pool.getName()) {
		return nil, fmt.Errorf("pool: %s is already registered", pool.getName())
	}

	r.registry.Add(pool.getName(), pool)
	// save to db

	return pool.getHandler(), nil
}

func (r *mpRegistry) RemoveMessagePool(pool_name string) error {
	if !r.registry.Contains(pool_name) {
		return fmt.Errorf("pool: %s is not registered", pool_name)
	}

	r.registry.Remove(pool_name)
	// remove from db

	return nil
}

func (r *mpRegistry) GetMessagePool(pool_name string) (MessagePoolHandler, error) {
	value, contains := r.registry.Get(pool_name)

	if !contains {
		return nil, fmt.Errorf("pool: %s is not registered", pool_name)
	}

	return value.getHandler(), nil
}
