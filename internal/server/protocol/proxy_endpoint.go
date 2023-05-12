package protocol

type ProxyEndpoint struct {
	Path     string `json:"path" binding:"required,startswith=/,min=2"`
	ProxyUrl string `json:"proxy_url" binding:"required,min=1"`
}
