package database

// bson names
const (
	STATIC_ENDPOINT_PATH_FIELD         = "path"
	STATIC_ENDPOINT_RESPONSE_FIELD     = "response"
	DYNAMIC_ENDPOINT_PATH_FIELD        = "path"
	DYNAMIC_ENDPOINT_SCRIPT_NAME_FIELD = "script_name"
	TASK_ID_FIELD                      = "task_id"
	MESSAGE_FIELD                      = "message"
)

type StaticEndpoint struct {
	Path     string `bson:"path" json:"path"`
	Response string `bson:"response" json:"response"`
}

type DynamicEndpoint struct {
	Path       string `bson:"path" json:"path"`
	ScriptName string `bson:"script_name" json:"script_name"`
}

type TaskMessage struct {
	TaskId  string `bson:"task_id"`
	Message string `bson:"message"`
}
