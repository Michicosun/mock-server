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

func (db *inmemoryDB) AddStaticEndpoint(path string, expected_response []byte) {
	db.static_routes.Add(path, string(expected_response))
}

func (db *inmemoryDB) RemoveStaticEndpoint(path string) {
	db.static_routes.Remove(path)
}

func (db *inmemoryDB) GetStaticEndpointResponse(path string) ([]byte, error) {
	response, ok := db.static_routes.Get(path)
	if !ok {
		return nil, errors.New("no such path")
	}

	return []byte(response), nil
}

func (db *inmemoryDB) PeekStaticEndpoint(path string) bool {
	return db.static_routes.Contains(path)
}
