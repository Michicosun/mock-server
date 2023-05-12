package protocol

type StaticEndpoint struct {
	Path             string `json:"path" binding:"required,startswith=/,min=2"`
	ExpectedResponse string `json:"expected_response" binding:"required,min=1"`
}
