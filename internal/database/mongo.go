package database

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

const databaseName = "mongo_storage"

type MongoStorage struct {
	client          *mongo.Client
	ctx             *context.Context
	staticEndpoints *staticEndpoints
}

var db = &MongoStorage{}

func (db *MongoStorage) Init(client *mongo.Client, ctx *context.Context) {
	db.client = client
	db.ctx = ctx
	db.staticEndpoints = &staticEndpoints{}
	db.staticEndpoints.init(client, db.ctx)
}

func AddStaticEndpoint(staticEndpoint StaticEndpoint) error {
	return db.staticEndpoints.addStaticEndpoint(staticEndpoint, db.ctx)
}

func RemoveStaticEndpoint(path string) error {
	return db.staticEndpoints.removeStaticEndpoint(path, db.ctx)
}

func GetStaticEndpointResponse(path string) (string, error) {
	return db.staticEndpoints.getStaticEndpointResponse(path, db.ctx)
}

func ListAllStaticEndpoints() ([]StaticEndpoint, error) {
	return db.staticEndpoints.listAllStaticEndpoints(db.ctx)
}
