package database

type StaticEndpoint struct {
	Path     string `bson:"path" json:"path"`
	Response string `bson:"response" json:"response"`
}

type DynamicEndpoint struct {
	Path       string `bson:"path" json:"path"`
	ScriptName string `bson:"script_name" json:"script_name"`
}
