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
	TASK_MESSAGES_COLLECTION     = "task_messages"
)

type MongoStorage struct {
	client           *mongo.Client
	staticEndpoints  *staticEndpoints
	dynamicEndpoints *dynamicEndpoints
	taskMessages     *taskMessages
}

var db = &MongoStorage{}

func (db *MongoStorage) init(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) error {
	db.client = client
	var err error
	db.staticEndpoints, err = createStaticEndpoints(ctx, client, cfg)
	if err != nil {
		return err
	}
	db.dynamicEndpoints, err = createDynamicEndpoints(ctx, client, cfg)
	if err != nil {
		return err
	}
	db.taskMessages, err = createTaskMessages(ctx, client, cfg)
	return err
}

func AddStaticEndpoint(ctx context.Context, staticEndpoint StaticEndpoint) error {
	return db.staticEndpoints.addStaticEndpoint(ctx, staticEndpoint)
}

func RemoveStaticEndpoint(ctx context.Context, path string) error {
	return db.staticEndpoints.removeStaticEndpoint(ctx, path)
}

func UpdateStaticEndpoint(ctx context.Context, staticEndpoint StaticEndpoint) error {
	return db.staticEndpoints.updateStaticEndpoint(ctx, staticEndpoint)
}

func GetStaticEndpointResponse(ctx context.Context, path string) (string, error) {
	return db.staticEndpoints.getStaticEndpointResponse(ctx, path)
}

func ListAllStaticEndpointPaths(ctx context.Context) ([]string, error) {
	return db.staticEndpoints.listAllStaticEndpointPaths(ctx)
}

func AddDynamicEndpoint(ctx context.Context, dynamicEndpoint DynamicEndpoint) error {
	return db.dynamicEndpoints.addDynamicEndpoint(ctx, dynamicEndpoint)
}

func RemoveDynamicEndpoint(ctx context.Context, path string) error {
	return db.dynamicEndpoints.removeDynamicEndpoint(ctx, path)
}

func UpdateDynamicEndpoint(ctx context.Context, dynamicEndpoint DynamicEndpoint) error {
	return db.dynamicEndpoints.updateDynamicEndpoint(ctx, dynamicEndpoint)
}

func GetDynamicEndpointScriptName(ctx context.Context, path string) (string, error) {
	return db.dynamicEndpoints.getDynamicEndpointScriptName(ctx, path)
}

func ListAllDynamicEndpointPaths(ctx context.Context) ([]string, error) {
	return db.dynamicEndpoints.listAllDynamicEndpointPaths(ctx)
}

func AddTaskMessages(ctx context.Context, taskMessages TaskMessage) error {
	return db.taskMessages.addTaskMessages(ctx, taskMessages)
}

func HasEndpoint(ctx context.Context, path string) (bool, error) {
	static, err1 := HasStaticEndpoint(ctx, path)
	if err1 != nil {
		return false, err1
	}
	dynamic, err2 := HasDynamicEndpoint(ctx, path)
	if err2 != nil {
		return false, err2
	}
	return static || dynamic, nil
}

func HasStaticEndpoint(ctx context.Context, path string) (bool, error) {
	_, err := GetStaticEndpointResponse(ctx, path)
	switch err {
	case nil:
		return true, nil
	case ErrNoSuchPath:
		return false, nil
	}
	return false, err
}

func HasDynamicEndpoint(ctx context.Context, path string) (bool, error) {
	_, err := GetDynamicEndpointScriptName(ctx, path)
	switch err {
	case nil:
		return true, nil
	case ErrNoSuchPath:
		return false, nil
	}
	return false, err
}
