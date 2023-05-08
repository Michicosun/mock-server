package database_test

import (
	"context"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"mock-server/internal/database"
	"testing"
)

var messagePoolsTests = []struct {
	testName  string
	cacheSize int
}{
	{"one elem in cache", 1},
	{"Some elems in cache", 3},
	{"inf cache", 0},
}

func comparePools(one *database.MessagePool, two *database.MessagePool) bool {
	return one.Name == two.Name && one.Broker == two.Broker
}

func TestMessagePools(t *testing.T) {
	for _, tt := range messagePoolsTests {
		t.Run(tt.testName, func(t *testing.T) {
			t.Setenv("CONFIG_PATH", "/configs/test_database_config.yaml")
			configs.SetConfigureForTestingFunc(func(cfg *configs.ServiceConfig) {
				cfg.Database.CacheSize = tt.cacheSize
			})
			control.Components.Start()
			defer control.Components.Stop()

			messagePools := []database.MessagePool{
				{Name: "pool1", Broker: "broker1", Config: []byte("{}")},
				{Name: "pool2", Broker: "broker2", Config: []byte("{}")},
				{Name: "pool3", Broker: "broker3", Config: []byte("{}")},
			}

			for _, messagePool := range messagePools {
				err := database.AddMessagePool(context.TODO(), messagePool)
				if err != nil {
					t.Error(err)
				}
			}

			for _, messagePool := range messagePools {
				err := database.AddMessagePool(context.TODO(), messagePool)
				if err != database.ErrDuplicateKey {
					t.Errorf("Expected database.ErrDuplicateKey")
				}
			}

			{
				err := database.RemoveMessagePool(context.TODO(), messagePools[1].Name)
				if err != nil {
					t.Error(err)
				}
				_, err = database.GetMessagePool(context.TODO(), messagePools[1].Name)
				if err != database.ErrNoSuchPool {
					t.Errorf("Expected ErrNoSuchPool")
				}
				err = database.AddMessagePool(context.TODO(), messagePools[1])
				if err != nil {
					t.Error(err)
				}
			}

			{
				messagePool, err := database.GetMessagePool(context.TODO(), messagePools[0].Name)
				if err != nil {
					t.Error(err)
				}
				if !comparePools(&messagePool, &messagePools[0]) {
					t.Errorf("res != expected: %s != %s", messagePool, messagePools[0])
				}
			}

			{
				for _, expectedMessagePool := range messagePools {
					messagePool, err := database.GetMessagePool(context.TODO(), expectedMessagePool.Name)
					if err != nil {
						t.Error(err)
					}
					if !comparePools(&messagePool, &expectedMessagePool) {
						t.Errorf("res != expected: %s != %s", messagePool, expectedMessagePool)
					}
				}
			}

		})
	}
}
