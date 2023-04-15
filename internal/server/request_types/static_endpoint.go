package requesttypes

type StaticEndpointPath struct {
	Path string `json:"path"`
}

type StaticEndpoint struct {
	Path             string `json:"path"`
	ExpectedResponse string `json:"expected_response"`
}
