package database_test

import (
	"fmt"
	"math/rand"
	"mock-server/internal/control"
	"mock-server/internal/database"
	"testing"
)

func comparePaths(paths []string, expected []database.StaticEndpoint) bool {
	if len(paths) != len(expected) {
		return false
	}
	for i := 0; i < len(paths); i++ {
		if paths[i] != expected[i].Path {
			return false
		}
	}
	return true
}

func TestStaticEndpoints(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_database_config.yaml")
	control.Components.Start()
	defer control.Components.Stop()

	endpoints := make([]database.StaticEndpoint, 0)
	endpoints = append(endpoints, database.StaticEndpoint{Path: "/one", Response: "one"})
	endpoints = append(endpoints, database.StaticEndpoint{Path: "/two", Response: "two"})
	endpoints = append(endpoints, database.StaticEndpoint{Path: "/three", Response: "three"})
	endpoints = append(endpoints, database.StaticEndpoint{Path: "/four", Response: "four"})
	endpoints = append(endpoints, database.StaticEndpoint{Path: "/five", Response: "five"})

	for _, endpoint := range endpoints {
		err := database.AddStaticEndpoint(endpoint)
		if err != nil {
			t.Errorf("AddStaticEndpoint return err: %s", err.Error())
		}
	}

	res, err := database.ListAllStaticEndpointPaths()
	if err != nil {
		t.Errorf("ListAllStaticEndpoints return err: %s", err.Error())
	}

	if !comparePaths(res, endpoints) {
		t.Errorf("res != expected: %s != %s", res, endpoints)
	}

	for _, endpoint := range endpoints {
		res, err := database.GetStaticEndpointResponse(endpoint.Path)
		if err != nil {
			t.Errorf("GetStaticEndpointResponse return err: %s", err.Error())
		}
		if res != endpoint.Response {
			t.Errorf("res != expected: %s != %s", res, endpoint.Response)
		}
	}

	for i := 0; i < 5; i++ {
		id := rand.Int() % (5 - i)
		err := database.RemoveStaticEndpoint(endpoints[id].Path)
		if err != nil {
			t.Errorf("RemoveStaticEndpoint return err: %s", err.Error())
		}
		endpoints = append(endpoints[:id], endpoints[id+1:]...)
		fmt.Println(len(endpoints))
		res, err := database.ListAllStaticEndpointPaths()
		if err != nil {
			t.Errorf("ListAllStaticEndpoints return err: %s", err.Error())
		}

		if !comparePaths(res, endpoints) {
			t.Errorf("res != expected: %s != %s", res, endpoints)
		}
	}
}
