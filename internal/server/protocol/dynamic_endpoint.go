package protocol

type DynamicEndpoint struct {
	Path string `json:"path" binding:"required,startswith=/,min=2"`
	Code string `json:"code" binding:"required,startswith=def func"`
}
