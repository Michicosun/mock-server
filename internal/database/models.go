package database

// bson names
const (
	// routes
	ROUTE_PATH_FIELD        = "path"
	ROUTE_TYPE_FIELD        = "type"
	ROUTE_RESPONSE_FIELD    = "response"
	ROUTE_SCRIPT_NAME_FIELD = "script_name"

	// endpoint types
	STATIC_ENDPOINT_TYPE  = "static_endpoint"
	PROXY_ENDPOINT_TYPE   = "proxy_endpoint"
	DYNAMIC_ENDPOINT_TYPE = "dynamic_endpoint"

	// task messages
	TASK_ID_FIELD = "task_id"
	MESSAGE_FIELD = "message"

	// esb records
	POOL_NAME_IN_FIELD       = "pool_name_in"
	POOL_NAME_OUT_FIELD      = "pool_name_out"
	MAPPER_SCRIPT_NAME_FIELD = "mapper_script_name"

	// message pool
	MESSAGE_POOL_NAME   = "name"
	MESSAGE_POOL_QUEUE  = "queue"
	MESSAGE_POOL_BROKER = "broker"
	MESSAGE_POOL_CONFIG = "config"
)

type Route struct {
	Path       string `bson:"path"`
	Type       string `bson:"type"`
	ScriptName string `bson:"script_name,omitempty"`
	Response   string `bson:"response,omitempty"`
	ProxyURL   string `bson:"proxy_url"`
}

type TaskMessage struct {
	TaskId  string `bson:"task_id"`
	Message string `bson:"message"`
}

type ESBRecord struct {
	PoolNameIn       string `bson:"pool_name_in"`
	PoolNameOut      string `bson:"pool_name_out"`
	MapperScriptName string `bson:"mapper_script_name"`
}

type MessagePool struct {
	Name   string `bson:"name"`
	Queue  string `bson:"queue"`
	Broker string `bson:"broker"`
	Config []byte `bson:"config"`
}
