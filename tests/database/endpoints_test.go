package database_test

import (
	"math/rand"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"mock-server/internal/database"
	"testing"
)

var endpointsTests = []struct {
	testName  string
	cacheSize int
}{
	{"one elem in cache", 1},
	{"Some elems in cache", 3},
	{"inf cache", 0},
}

func compareStaticEndpointPaths(paths []string, expected []database.StaticEndpoint) bool {
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
	for _, tt := range endpointsTests {
		t.Run(tt.testName, func(t *testing.T) {
			t.Setenv("CONFIG_PATH", "/configs/test_database_config.yaml")
			configs.SetConfigureForTestingFunc(func(cfg *configs.ServiceConfig) {
				cfg.Database.CacheSize = tt.cacheSize
			})
			control.Components.Start()
			defer control.Components.Stop()

			endpoints := make([]database.StaticEndpoint, 0)
			endpoints = append(endpoints, database.StaticEndpoint{Path: "/one", Response: "one"})
			endpoints = append(endpoints, database.StaticEndpoint{Path: "/two", Response: "two"})
			endpoints = append(endpoints, database.StaticEndpoint{Path: "/three", Response: "three"})
			endpoints = append(endpoints, database.StaticEndpoint{Path: "/four", Response: "four"})
			endpoints = append(endpoints, database.StaticEndpoint{Path: "/five", Response: "five"})

			for _, endpoint := range endpoints {
				if err := database.AddStaticEndpoint(endpoint); err != nil {
					t.Errorf("AddStaticEndpoint return err: %s", err.Error())
				}
			}

			// Check that we store only unique elems
			for _, endpoint := range endpoints {
				if err := database.AddStaticEndpoint(endpoint); err != nil {
					t.Errorf("AddStaticEndpoint return err: %s", err.Error())
				}
			}

			res, err := database.ListAllStaticEndpointPaths()
			if err != nil {
				t.Errorf("ListAllStaticEndpointPaths return err: %s", err.Error())
			}

			if !compareStaticEndpointPaths(res, endpoints) {
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
				if err := database.RemoveStaticEndpoint(endpoints[id].Path); err != nil {
					t.Errorf("RemoveStaticEndpoint return err: %s", err.Error())
				}
				endpoints = append(endpoints[:id], endpoints[id+1:]...)
				res, err := database.ListAllStaticEndpointPaths()
				if err != nil {
					t.Errorf("ListAllStaticEndpoints return err: %s", err.Error())
				}

				if !compareStaticEndpointPaths(res, endpoints) {
					t.Errorf("res != expected: %s != %s", res, endpoints)
				}
			}

			database.AddStaticEndpoint(database.StaticEndpoint{
				Path:     "/path",
				Response: "one",
			})
			database.AddStaticEndpoint(database.StaticEndpoint{
				Path:     "/path",
				Response: "two",
			})
			response, err := database.GetStaticEndpointResponse("/path")
			if err != nil {
				t.Errorf("GetStaticEndpointResponse return err: %s", err.Error())
			}
			if response != "one" {
				t.Errorf("response != expected: %s != one", response)
			}
		})
	}
}

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
	for _, tt := range endpointsTests {
		t.Run(tt.testName, func(t *testing.T) {
			t.Setenv("CONFIG_PATH", "/configs/test_database_config.yaml")
			configs.SetConfigureForTestingFunc(func(cfg *configs.ServiceConfig) {
				cfg.Database.CacheSize = tt.cacheSize
			})
			control.Components.Start()
			defer control.Components.Stop()

			endpoints := make([]database.DynamicEndpoint, 0)
			endpoints = append(endpoints, database.DynamicEndpoint{Path: "/one", ScriptName: "one"})
			endpoints = append(endpoints, database.DynamicEndpoint{Path: "/two", ScriptName: "two"})
			endpoints = append(endpoints, database.DynamicEndpoint{Path: "/three", ScriptName: "three"})
			endpoints = append(endpoints, database.DynamicEndpoint{Path: "/four", ScriptName: "four"})
			endpoints = append(endpoints, database.DynamicEndpoint{Path: "/five", ScriptName: "five"})

			for _, endpoint := range endpoints {
				if err := database.AddDynamicEndpoint(endpoint); err != nil {
					t.Errorf("AddDynamicEndpoint return err: %s", err.Error())
				}
			}

			for _, endpoint := range endpoints {
				if err := database.AddDynamicEndpoint(endpoint); err != nil {
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
				if err := database.RemoveDynamicEndpoint(endpoints[id].Path); err != nil {
					t.Errorf("RemoveDynamicEndpoint return err: %s", err.Error())
				}
				endpoints = append(endpoints[:id], endpoints[id+1:]...)
				res, err := database.ListAllDynamicEndpointPaths()
				if err != nil {
					t.Errorf("ListAllDynamicEndpointPaths return err: %s", err.Error())
				}

				if !compareDynamicEndpointPaths(res, endpoints) {
					t.Errorf("res != expected: %s != %s", res, endpoints)
				}
			}
		})
	}
}
