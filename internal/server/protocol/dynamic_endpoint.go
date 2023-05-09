package protocol

type DynamicEndpoint struct {
	Path string `json:"path"`
	Code string `json:"code"`
}

type DynamicEndpointCodeQuery struct {
	Path string `json:"path"`
}
