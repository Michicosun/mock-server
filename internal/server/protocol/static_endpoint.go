package protocol

type StaticEndpoint struct {
	Path             string `json:"path"`
	ExpectedResponse string `json:"expected_response"`
}
