package database

type Database interface {
	// static endpoints
	AddStaticEndpoint(path string, expected_response []byte)
	RemoveStaticEndpoint(path string)
	GetStaticEndpointResponse(path string) ([]byte, error)
	PeekStaticEndpoint(path string) bool
}

func NewInmemoryDatabase() Database {
	return newInmemoryDatabase()
}
