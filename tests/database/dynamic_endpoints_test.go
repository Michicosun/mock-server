package database_test

import (
	"fmt"
	"math/rand"
	"mock-server/internal/control"
	"mock-server/internal/database"
	"testing"
)

func compareDynamicEndpointPaths(paths []string, expected []database.DynamicEndpoint) bool {
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

func TestDynamicEndpoints(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_database_config.yaml")
	control.Components.Start()
	defer control.Components.Stop()

	endpoints := make([]database.DynamicEndpoint, 0)
	endpoints = append(endpoints, database.DynamicEndpoint{Path: "/one", ScriptName: "one"})
	endpoints = append(endpoints, database.DynamicEndpoint{Path: "/two", ScriptName: "two"})
	endpoints = append(endpoints, database.DynamicEndpoint{Path: "/three", ScriptName: "three"})
	endpoints = append(endpoints, database.DynamicEndpoint{Path: "/four", ScriptName: "four"})
	endpoints = append(endpoints, database.DynamicEndpoint{Path: "/five", ScriptName: "five"})

	for _, endpoint := range endpoints {
		err := database.AddDynamicEndpoint(endpoint)
		if err != nil {
			t.Errorf("AddDynamicEndpoint return err: %s", err.Error())
		}
	}

	res, err := database.ListAllDynamicEndpointPaths()
	if err != nil {
		t.Errorf("ListAllDynamicEndpointPaths return err: %s", err.Error())
	}

	if !compareDynamicEndpointPaths(res, endpoints) {
		t.Errorf("res != expected: %s != %s", res, endpoints)
	}

	for _, endpoint := range endpoints {
		res, err := database.GetDynamicEndpointScriptName(endpoint.Path)
		if err != nil {
			t.Errorf("GetDynamicEndpointScriptName return err: %s", err.Error())
		}
		if res != endpoint.ScriptName {
			t.Errorf("res != expected: %s != %s", res, endpoint.ScriptName)
		}
	}

	for i := 0; i < 5; i++ {
		id := rand.Int() % (5 - i)
		err := database.RemoveDynamicEndpoint(endpoints[id].Path)
		if err != nil {
			t.Errorf("RemoveDynamicEndpoint return err: %s", err.Error())
		}
		endpoints = append(endpoints[:id], endpoints[id+1:]...)
		fmt.Println(len(endpoints))
		res, err := database.ListAllDynamicEndpointPaths()
		if err != nil {
			t.Errorf("ListAllDynamicEndpointPaths return err: %s", err.Error())
		}

		if !compareDynamicEndpointPaths(res, endpoints) {
			t.Errorf("res != expected: %s != %s", res, endpoints)
		}
	}
}
