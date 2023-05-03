package database

type StaticEndpoint struct {
	Path     string `bson:"path" json:"path"`
	Response string `bson:"response,omitempty" json:"response"`
}
