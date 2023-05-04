package database

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

const databaseName = "mongo_storage"

type MongoStorage struct {
	client           *mongo.Client
	ctx              context.Context
	staticEndpoints  *staticEndpoints
	dynamicEndpoints *dynamicEndpoints
}

var db = &MongoStorage{}

func (db *MongoStorage) init(ctx context.Context, client *mongo.Client) {
	db.client = client
	db.ctx = ctx
	db.staticEndpoints = &staticEndpoints{}
	db.staticEndpoints.init(db.ctx, client)
	db.dynamicEndpoints = &dynamicEndpoints{}
	db.dynamicEndpoints.init(db.ctx, client)
}

func AddStaticEndpoint(staticEndpoint StaticEndpoint) error {
	return db.staticEndpoints.addStaticEndpoint(db.ctx, staticEndpoint)
}

func RemoveStaticEndpoint(path string) error {
	return db.staticEndpoints.removeStaticEndpoint(db.ctx, path)
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

func GetDynamicEndpointScriptName(path string) (string, error) {
	return db.dynamicEndpoints.getDynamicEndpointScriptName(db.ctx, path)
}

func ListAllDynamicEndpointPaths() ([]string, error) {
	return db.dynamicEndpoints.listAllDynamicEndpointPaths(db.ctx)
}
