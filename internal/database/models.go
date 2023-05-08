package database

// bson names
const (
	STATIC_ENDPOINT_PATH_FIELD         = "path"
	STATIC_ENDPOINT_RESPONSE_FIELD     = "response"
	DYNAMIC_ENDPOINT_PATH_FIELD        = "path"
	DYNAMIC_ENDPOINT_SCRIPT_NAME_FIELD = "script_name"
	TASK_ID_FIELD                      = "task_id"
	MESSAGE_FIELD                      = "message"
	POOL_NAME_IN_FIELD                 = "pool_name_in"
	POOL_NAME_OUT_FIELD                = "pool_name_out"
	MAPPER_SCRIPT_NAME_FIELD           = "mapper_script_name"
	MESSAGE_POOL_NAME                  = "name"
	MESSAGE_POOL_BROKER                = "broker"
	MESSAGE_POOL_CONFIG                = "config"
)

type StaticEndpoint struct {
	Path     string `bson:"path"`
	Response string `bson:"response"`
}

type DynamicEndpoint struct {
	Path       string `bson:"path"`
	ScriptName string `bson:"script_name"`
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
	Broker string `bson:"broker"`
	Config []byte `bson:"config"`
}
