package database

import (
	"errors"
	"mock-server/internal/util"
)

type inmemoryDB struct {
	static_routes util.SyncMap[string, string]
}

func newInmemoryDatabase() *inmemoryDB {
	db := inmemoryDB{
		static_routes: util.NewSyncMap[string, string](),
	}

	return &db
}

func (db *inmemoryDB) AddStaticEndpoint(path string, expected_response string) {
	db.static_routes.Add(path, string(expected_response))
}

func (db *inmemoryDB) RemoveStaticEndpoint(path string) {
	db.static_routes.Remove(path)
}

func (db *inmemoryDB) GetStaticEndpointResponse(path string) (string, error) {
	response, ok := db.static_routes.Get(path)
	if !ok {
		return "", errors.New("no such path")
	}

	return response, nil
}

func (db *inmemoryDB) ListAllStaticEndpoints() []string {
	return db.static_routes.GetAllKeys()
}

func (db *inmemoryDB) HasStaticEndpoint(path string) bool {
	return db.static_routes.Contains(path)
}
