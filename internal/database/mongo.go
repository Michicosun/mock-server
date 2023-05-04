package database

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

const databaseName = "mongo_storage"

type MongoStorage struct {
	client          *mongo.Client
	ctx             context.Context
	staticEndpoints *staticEndpoints
}

var db = &MongoStorage{}

func (db *MongoStorage) init(ctx context.Context, client *mongo.Client) {
	db.client = client
	db.ctx = ctx
	db.staticEndpoints = &staticEndpoints{}
	db.staticEndpoints.init(db.ctx, client)
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
