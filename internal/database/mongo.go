package database

import (
	"context"
	"mock-server/internal/configs"

	"go.mongodb.org/mongo-driver/mongo"
)

const (
	DATABASE_NAME                = "mongo_storage"
	STATIC_ENDPOINTS_COLLECTION  = "static_endpoints"
	DYNAMIC_ENDPOINTS_COLLECTION = "dynamic_endpoints"
)

type MongoStorage struct {
	client           *mongo.Client
	ctx              context.Context
	staticEndpoints  *staticEndpoints
	dynamicEndpoints *dynamicEndpoints
}

var db = &MongoStorage{}

func (db *MongoStorage) init(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) error {
	db.client = client
	db.ctx = ctx
	var err error
	db.staticEndpoints, err = createStaticEndpoints(db.ctx, client, cfg)
	if err != nil {
		return err
	}
	db.dynamicEndpoints, err = createDynamicEndpoints(db.ctx, client, cfg)
	if err != nil {
		return err
	}
	return nil
}

func AddStaticEndpoint(staticEndpoint StaticEndpoint) error {
	return db.staticEndpoints.addStaticEndpoint(db.ctx, staticEndpoint)
}

func RemoveStaticEndpoint(path string) error {
	return db.staticEndpoints.removeStaticEndpoint(db.ctx, path)
}

func UpdateStaticEndpoint(staticEndpoint StaticEndpoint) error {
	return db.staticEndpoints.updateStaticEndpoint(db.ctx, staticEndpoint)
}

func GetStaticEndpointResponse(path string) (string, error) {
	return db.staticEndpoints.getStaticEndpointResponse(db.ctx, path)
}

func ListAllStaticEndpointPaths() ([]string, error) {
	return db.staticEndpoints.listAllStaticEndpointPaths(db.ctx)
}

func AddDynamicEndpoint(dynamicEndpoint DynamicEndpoint) error {
	return db.dynamicEndpoints.addDynamicEndpoint(db.ctx, dynamicEndpoint)
}

func RemoveDynamicEndpoint(path string) error {
	return db.dynamicEndpoints.removeDynamicEndpoint(db.ctx, path)
}

func UpdateDynamicEndpoint(dynamicEndpoint DynamicEndpoint) error {
	return db.dynamicEndpoints.updateDynamicEndpoint(db.ctx, dynamicEndpoint)
}

func GetDynamicEndpointScriptName(path string) (string, error) {
	return db.dynamicEndpoints.getDynamicEndpointScriptName(db.ctx, path)
}

func ListAllDynamicEndpointPaths() ([]string, error) {
	return db.dynamicEndpoints.listAllDynamicEndpointPaths(db.ctx)
}

func HasEndpoint(path string) (bool, error) {
	static, err1 := HasStaticEndpoint(path)
	if err1 != nil {
		return false, err1
	}
	dynamic, err2 := HasDynamicEndpoint(path)
	if err2 != nil {
		return false, err2
	}
	return static || dynamic, nil
}

func HasStaticEndpoint(path string) (bool, error) {
	_, err := GetStaticEndpointResponse(path)
	switch err {
	case nil:
		return true, nil
	case ErrNoSuchPath:
		return false, nil
	}
	return false, err
}

func HasDynamicEndpoint(path string) (bool, error) {
	_, err := GetDynamicEndpointScriptName(path)
	switch err {
	case nil:
		return true, nil
	case ErrNoSuchPath:
		return false, nil
	}
	return false, err
}
