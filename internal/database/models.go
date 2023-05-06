package database

// bson names
const (
	STATIC_ENDPOINT_PATH         = "path"
	STATIC_ENDPOINT_RESPONSE     = "response"
	DYNAMIC_ENDPOINT_PATH        = "path"
	DYNAMIC_ENDPOINT_SCRIPT_NAME = "script_name"
)

type StaticEndpoint struct {
	Path     string `bson:"path" json:"path"`
	Response string `bson:"response" json:"response"`
}

type DynamicEndpoint struct {
	Path       string `bson:"path" json:"path"`
	ScriptName string `bson:"script_name" json:"script_name"`
}
