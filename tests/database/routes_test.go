package database_test

import (
	"context"
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

func compareRoutesPaths(paths []string, expected []database.Route) bool {
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

func TestRoutes(t *testing.T) {
	for _, tt := range endpointsTests {
		t.Run(tt.testName, func(t *testing.T) {
			t.Setenv("CONFIG_PATH", "/configs/test_database_config.yaml")
			configs.SetConfigureForTestingFunc(func(cfg *configs.ServiceConfig) {
				cfg.Database.CacheSize = tt.cacheSize
			})
			control.Components.Start()
			defer control.Components.Stop()

			staticRoutes := []database.Route{
				{Path: "/one", Type: database.STATIC_ENDPOINT_TYPE, Response: "one"},
				{Path: "/two", Type: database.STATIC_ENDPOINT_TYPE, Response: "two"},
				{Path: "/three", Type: database.STATIC_ENDPOINT_TYPE, Response: "three"},
			}

			dynamicRoutes := []database.Route{
				{Path: "/four", Type: database.DYNAMIC_ENDPOINT_TYPE, ScriptName: "four"},
				{Path: "/five", Type: database.DYNAMIC_ENDPOINT_TYPE, ScriptName: "five"},
				{Path: "/six", Type: database.DYNAMIC_ENDPOINT_TYPE, ScriptName: "six"},
			}

			for _, route := range staticRoutes {
				if err := database.AddStaticEndpoint(context.TODO(), route.Path, route.Response); err != nil {
					t.Errorf("AddRoute return err: %s", err.Error())
				}
			}

			for _, route := range dynamicRoutes {
				if err := database.AddDynamicEndpoint(context.TODO(), route.Path, route.ScriptName); err != nil {
					t.Errorf("AddRoute return err: %s", err.Error())
				}
			}

			{
				// Check that we store only unique elems
				for _, route := range staticRoutes {
					if err := database.AddStaticEndpoint(context.TODO(), route.Path, route.Response); err != database.ErrDuplicateKey {
						t.Errorf("AddRoute shoud return ErrDuplicateKey")
					}
					if err := database.AddDynamicEndpoint(context.TODO(), route.Path, route.ScriptName); err != database.ErrDuplicateKey {
						t.Errorf("AddRoute shoud return ErrDuplicateKey")
					}
				}

				for _, route := range dynamicRoutes {
					if err := database.AddStaticEndpoint(context.TODO(), route.Path, route.Response); err != database.ErrDuplicateKey {
						t.Errorf("AddRoute shoud return ErrDuplicateKey")
					}
					if err := database.AddDynamicEndpoint(context.TODO(), route.Path, route.ScriptName); err != database.ErrDuplicateKey {
						t.Errorf("AddRoute shoud return ErrDuplicateKey")
					}
				}
			}

			{
				res, err := database.ListAllStaticEndpointPaths(context.TODO())
				if err != nil {
					t.Error(err)
				}

				if !compareRoutesPaths(res, staticRoutes) {
					t.Errorf("res != expected: %s != %s", res, staticRoutes)
				}
			}

			{
				res, err := database.ListAllDynamicEndpointPaths(context.TODO())
				if err != nil {
					t.Error(err)
				}

				if !compareRoutesPaths(res, dynamicRoutes) {
					t.Errorf("res != expected: %s != %s", res, dynamicRoutes)
				}
			}

			{
				for _, route := range staticRoutes {
					res, err := database.GetStaticEndpointResponse(context.TODO(), route.Path)
					if err != nil {
						t.Error(err)
					}
					if res != route.Response {
						t.Errorf("res != expected: %s != %s", res, route.Response)
					}
				}
				for _, route := range dynamicRoutes {
					res, err := database.GetDynamicEndpointScriptName(context.TODO(), route.Path)
					if err != nil {
						t.Error(err)
					}
					if res != route.ScriptName {
						t.Errorf("res != expected: %s != %s", res, route.ScriptName)
					}
				}
			}

			{
				for _, route := range dynamicRoutes {
					if _, err := database.GetStaticEndpointResponse(context.TODO(), route.Path); err != database.ErrBadRouteType {
						t.Errorf("Expected ErrBadRouteType")
					}
				}
				for _, route := range staticRoutes {
					if _, err := database.GetDynamicEndpointScriptName(context.TODO(), route.Path); err != database.ErrBadRouteType {
						t.Errorf("Expected ErrBadRouteType")
					}
				}
			}

			{
				for i := 0; i < 3; i++ {
					id := rand.Int() % (3 - i)
					if err := database.RemoveStaticEndpoint(context.TODO(), staticRoutes[id].Path); err != nil {
						t.Errorf("RemoveRoute return err: %s", err.Error())
					}
					staticRoutes = append(staticRoutes[:id], staticRoutes[id+1:]...)
					res, err := database.ListAllStaticEndpointPaths(context.TODO())
					if err != nil {
						t.Errorf("ListAllRoutes return err: %s", err.Error())
					}
					if !compareRoutesPaths(res, staticRoutes) {
						t.Errorf("res != expected: %+q != %+q", res, staticRoutes)
					}
				}
			}

			{
				for i := 0; i < 3; i++ {
					id := rand.Int() % (3 - i)
					if err := database.RemoveDynamicEndpoint(context.TODO(), dynamicRoutes[id].Path); err != nil {
						t.Errorf("RemoveRoute return err: %s", err.Error())
					}
					dynamicRoutes = append(dynamicRoutes[:id], dynamicRoutes[id+1:]...)
					res, err := database.ListAllDynamicEndpointPaths(context.TODO())
					if err != nil {
						t.Errorf("ListAllRoutes return err: %s", err.Error())
					}
					if !compareRoutesPaths(res, dynamicRoutes) {
						t.Errorf("res != expected: %+q != %+q", res, dynamicRoutes)
					}
				}
			}

			{
				if err := database.AddStaticEndpoint(context.TODO(), "/path", "one"); err != nil {
					t.Errorf("AddRoute return err: %s", err.Error())
				}
				if err := database.AddDynamicEndpoint(context.TODO(), "/path", "two"); err != database.ErrDuplicateKey {
					t.Errorf("AddRoute should return ErrDuplicateKey")
				}
				if err := database.AddStaticEndpoint(context.TODO(), "/path", "two"); err != database.ErrDuplicateKey {
					t.Errorf("AddRoute should return ErrDuplicateKey")
				}
				response, err := database.GetStaticEndpointResponse(context.TODO(), "/path")
				if err != nil {
					t.Errorf("GetRouteResponse return err: %s", err.Error())
				}
				if response != "one" {
					t.Errorf("response != expected: %s != one", response)
				}
			}

			{
				if err := database.UpdateDynamicEndpoint(context.TODO(), "/path", "two"); err != database.ErrNoSuchPath {
					t.Errorf("UpdateDynamicEndpoint shoud return ErrNoSuchPath, but returns %s", err)
				}
				if err := database.UpdateStaticEndpoint(context.TODO(), "/path", "two"); err != nil {
					t.Error(err)
				}
				response, err := database.GetStaticEndpointResponse(context.TODO(), "/path")
				if err != nil {
					t.Error(err)
				}
				if response != "two" {
					t.Errorf("response != expected: %s != two", response)
				}
			}
		})
	}
}