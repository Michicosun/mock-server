package protocol

type ProxyEndpoint struct {
	Path     string `json:"path"`
	ProxyUrl string `json:"proxy_url"`
}
