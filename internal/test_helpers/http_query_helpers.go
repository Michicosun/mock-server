package helpers

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func DoGet(url string, t *testing.T) (int, []byte) {
	resp, err := http.Get(url)
	if err != nil {
		t.Error(err)
		return 0, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		return 0, nil
	}

	return resp.StatusCode, body
}

func DoGetWithBody(url string, content []byte, t *testing.T) (int, []byte) {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(content))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Error(err)
		return 0, nil
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return 0, nil
	}

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Error(err)
		return 0, nil
	}

	return resp.StatusCode, body
}

func DoPost(url string, content []byte, t *testing.T) int {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(content))
	if err != nil {
		t.Error(err)
		return 0
	}

	return resp.StatusCode
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
