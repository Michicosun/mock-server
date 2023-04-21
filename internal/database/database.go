package database

import "mock-server/internal/util"

var DB Database = newInmemoryDatabase()

type Database interface {
	// static endpoints
	AddStaticEndpoint(path string, expected_response string)
	RemoveStaticEndpoint(path string)
	GetStaticEndpointResponse(path string) (string, error)
	ListAllStaticEndpoints() []string
	HasStaticEndpoint(path string) bool

	// brokers
	GetReadMessagesCollection() *util.SyncMap[string, interface{}]
	GetWriteMessagesCollection() *util.SyncMap[string, interface{}]

	// common
	Drop()
}
