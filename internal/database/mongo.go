package database

import (
	"context"
	"fmt"
	"mock-server/internal/configs"

	"go.mongodb.org/mongo-driver/mongo"
)

const (
	DATABASE_NAME            = "mongo_storage"
	ROUTES_COLLECTION        = "routes"
	TASK_MESSAGES_COLLECTION = "task_messages"
	ESB_RECORDS_COLLECTION   = "esb_records"
	MESSAGE_POOLS_COLLECTION = "message_pools"
)

type MongoStorage struct {
	client       *mongo.Client
	routes       *routes
	taskMessages *taskMessages
	esbRecords   *esbRecords
	messagePools *messagePools
}

var db = &MongoStorage{}

func (db *MongoStorage) init(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) error {
	db.client = client
	var err error
	db.routes, err = createRoutes(ctx, client, cfg)
	if err != nil {
		return err
	}
	db.taskMessages, err = createTaskMessages(ctx, client, cfg)
	if err != nil {
		return err
	}
	db.esbRecords, err = createESBRecords(ctx, client, cfg)
	if err != nil {
		return err
	}
	db.messagePools, err = createMessagePools(ctx, client, cfg)
	if err != nil {
		return err
	}
	return nil
}

func AddStaticEndpoint(ctx context.Context, path string, response string) error {
	return db.routes.addRoute(ctx, Route{
		Path:     path,
		Type:     STATIC_ENDPOINT_TYPE,
		Response: response,
	})
}

func RemoveStaticEndpoint(ctx context.Context, path string) error {
	return db.routes.removeRoute(ctx, path)
}

func UpdateStaticEndpoint(ctx context.Context, path string, response string) error {
	return db.routes.updateRoute(ctx, Route{
		Path:     path,
		Type:     STATIC_ENDPOINT_TYPE,
		Response: response,
	})
}

func GetStaticEndpointResponse(ctx context.Context, path string) (string, error) {
	route, err := db.routes.getRoute(ctx, path)
	if err != nil {
		return "", err
	}
	if route.Type != STATIC_ENDPOINT_TYPE {
		return "", ErrBadRouteType
	}
	return route.Response, nil
}

func ListAllStaticEndpointPaths(ctx context.Context) ([]string, error) {
	return db.routes.listAllRoutesPathsWithType(ctx, STATIC_ENDPOINT_TYPE)
}

func AddProxyEndpoint(ctx context.Context, path string, proxyUrl string) error {
	return db.routes.addRoute(ctx, Route{
		Path:     path,
		Type:     PROXY_ENDPOINT_TYPE,
		ProxyURL: proxyUrl,
	})
}

func RemoveProxyEndpoint(ctx context.Context, path string) error {
	return db.routes.removeRoute(ctx, path)
}

func UpdateProxyEndpoint(ctx context.Context, path string, proxyUrl string) error {
	return db.routes.updateRoute(ctx, Route{
		Path:     path,
		Type:     PROXY_ENDPOINT_TYPE,
		ProxyURL: proxyUrl,
	})
}

func GetProxyEndpointProxyUrl(ctx context.Context, path string) (string, error) {
	route, err := db.routes.getRoute(ctx, path)
	if err != nil {
		return "", err
	}
	if route.Type != PROXY_ENDPOINT_TYPE {
		return "", ErrBadRouteType
	}
	return route.ProxyURL, nil
}

func ListAllProxyEndpointPaths(ctx context.Context) ([]string, error) {
	return db.routes.listAllRoutesPathsWithType(ctx, PROXY_ENDPOINT_TYPE)
}

func AddDynamicEndpoint(ctx context.Context, path string, scriptName string) error {
	return db.routes.addRoute(ctx, Route{
		Path:       path,
		Type:       DYNAMIC_ENDPOINT_TYPE,
		ScriptName: scriptName,
	})
}

func RemoveDynamicEndpoint(ctx context.Context, path string) error {
	return db.routes.removeRoute(ctx, path)
}

func UpdateDynamicEndpoint(ctx context.Context, path string, scriptName string) error {
	return db.routes.updateRoute(ctx, Route{
		Path:       path,
		Type:       DYNAMIC_ENDPOINT_TYPE,
		ScriptName: scriptName,
	})
}

func GetDynamicEndpointScriptName(ctx context.Context, path string) (string, error) {
	route, err := db.routes.getRoute(ctx, path)
	if err != nil {
		return "", err
	}
	if route.Type != DYNAMIC_ENDPOINT_TYPE {
		return "", ErrBadRouteType
	}
	return route.ScriptName, nil
}

func GetRoute(ctx context.Context, path string) (Route, error) {
	return db.routes.getRoute(ctx, path)
}

func ListAllDynamicEndpointPaths(ctx context.Context) ([]string, error) {
	return db.routes.listAllRoutesPathsWithType(ctx, DYNAMIC_ENDPOINT_TYPE)
}

func AddTaskMessage(ctx context.Context, taskMessage TaskMessage) error {
	return db.taskMessages.addTaskMessage(ctx, taskMessage)
}

func GetTaskMessages(ctx context.Context, taskId string) ([]string, error) {
	return db.taskMessages.getTaskMessages(ctx, taskId)
}

func AddESBRecord(ctx context.Context, esbRecord ESBRecord) error {
	return db.esbRecords.addESBRecord(ctx, esbRecord)
}

func RemoveESBRecord(ctx context.Context, poolNameIn string) error {
	return db.esbRecords.removeESBRecord(ctx, poolNameIn)
}

func GetESBRecord(ctx context.Context, poolNameIn string) (ESBRecord, error) {
	return db.esbRecords.getESBRecord(ctx, poolNameIn)
}

func ListESBRecords(ctx context.Context) ([]ESBRecord, error) {
	return db.esbRecords.listESBRecords(ctx)
}

func AddMessagePool(ctx context.Context, messagePool MessagePool) error {
	return db.messagePools.addMessagePool(ctx, messagePool)
}

func RemoveMessagePool(ctx context.Context, name string) error {
	return db.messagePools.removeMessagePool(ctx, name)
}

func GetMessagePool(ctx context.Context, name string) (MessagePool, error) {
	return db.messagePools.getMessagePool(ctx, name)
}

func ListMessagePools(ctx context.Context) ([]MessagePool, error) {
	return db.messagePools.listMessagePools(ctx)
}

func GetMessagePoolReadMessages(ctx context.Context, messagePool MessagePool) ([]string, error) {
	poolTasksId := fmt.Sprintf("%s:%s:%s:read", messagePool.Broker, messagePool.Name, messagePool.Queue)

	return GetTaskMessages(ctx, poolTasksId)
}

func GetMessagePoolWriteMessages(ctx context.Context, messagePool MessagePool) ([]string, error) {
	poolTasksId := fmt.Sprintf("%s:%s:%s:write", messagePool.Broker, messagePool.Name, messagePool.Queue)

	return GetTaskMessages(ctx, poolTasksId)
}
