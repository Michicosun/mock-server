package server_test

import (
	"bytes"
	"fmt"
	"io"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"net/http"
	"testing"
)

func TestServerSimple(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)

	url := endpoint + "/api/ping"
	code, body := DoGet(url, t)

	if code != 200 {
		t.Errorf("ping status code != 200")
	}

	if !bytes.Equal(body, []byte(`"Ping yourself, I have another work!"`)) {
		t.Errorf(`ping: %s != "Ping yourself, I have another work!\n"`, body)
	}
}

func TestServerSameNamespaceForEndpoints(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	staticApiEndpoint := endpoint + "/api/routes/static"
	dynamicApiEndpoint := endpoint + "/api/routes/dynamic"

	requestBody := []byte(`{
		"path": "/test_url",
		"expected_response": "hello"
	}`)
	testBodyScript := []byte(`{
		"path": "/test_url",
		"code": "def func(A, B, C):\n    pass"
	}`)

	// post static endpoint than try to post same static and dynamic
	code, _ := DoPost(staticApiEndpoint, requestBody, t)
	if code != 200 {
		t.Errorf("create route failed: expected 200 != %d", code)
	}
	code, _ = DoPost(dynamicApiEndpoint, testBodyScript, t)
	if code != 409 {
		t.Errorf("expected to receive conflict: expected 409 != %d", code)
	}
	code, _ = DoPost(staticApiEndpoint, requestBody, t)
	if code != 409 {
		t.Errorf("expected to receive conflict: expected 409 != %d", code)
	}

	// wipe
	code = DoDelete(staticApiEndpoint+"?path=/test_url", t)
	if code != 204 {
		t.Errorf("expected to be possible to delete existing endpoint")
	}

	// same as first test but fisrt post dynamic
	code, _ = DoPost(dynamicApiEndpoint, testBodyScript, t)
	if code != 200 {
		t.Errorf("create route failed: expected 200 != %d", code)
	}
	code, _ = DoPost(staticApiEndpoint, requestBody, t)
	if code != 409 {
		t.Errorf("expected to receive conflict: expected 409 != %d", code)
	}
	code, _ = DoPost(dynamicApiEndpoint, testBodyScript, t)
	if code != 409 {
		t.Errorf("expected to receive conflict: expected 409 != %d", code)
	}
}

func DoGet(url string, t *testing.T) (int, []byte) {
	resp, err := http.Get(url)
	if err != nil {
		t.Error(err)
		return 0, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		return 0, nil
	}

	return resp.StatusCode, body
}

func DoPost(url string, content []byte, t *testing.T) (int, []byte) {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(content))
	if err != nil {
		t.Error(err)
		return 0, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		return 0, nil
	}

	return resp.StatusCode, body
}

func DoPut(url string, content []byte, t *testing.T) int {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(content))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Error(err)
		return 0
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return 0
	}

	defer resp.Body.Close()

	return resp.StatusCode
}

func DoDelete(url string, t *testing.T) int {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Error(err)
		return 0
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return 0
	}

	defer resp.Body.Close()

	return resp.StatusCode
}
