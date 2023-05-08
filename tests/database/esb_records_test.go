package database_test

import (
	"context"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"mock-server/internal/database"
	"testing"
)

var esbRecordsTests = []struct {
	testName  string
	cacheSize int
}{
	{"one elem in cache", 1},
	{"Some elems in cache", 3},
	{"inf cache", 0},
}

func TestESBRecords(t *testing.T) {
	for _, tt := range esbRecordsTests {
		t.Run(tt.testName, func(t *testing.T) {
			t.Setenv("CONFIG_PATH", "/configs/test_database_config.yaml")
			configs.SetConfigureForTestingFunc(func(cfg *configs.ServiceConfig) {
				cfg.Database.CacheSize = tt.cacheSize
			})
			control.Components.Start()
			defer control.Components.Stop()

			esbRecords := []database.ESBRecord{
				{PoolNameIn: "test_pool_in_1", PoolNameOut: "test_pool_out_1", MapperScriptName: "mapper_script_name1"},
				{PoolNameIn: "test_pool_in_2", PoolNameOut: "test_pool_out_2", MapperScriptName: "mapper_script_name2"},
				{PoolNameIn: "test_pool_in_3", PoolNameOut: "test_pool_out_3", MapperScriptName: "mapper_script_name3"},
			}

			for _, esbRecord := range esbRecords {
				err := database.AddESBRecord(context.TODO(), esbRecord)
				if err != nil {
					t.Error(err)
				}
			}

			for _, esbRecord := range esbRecords {
				err := database.AddESBRecord(context.TODO(), esbRecord)
				if err != database.ErrDuplicateKey {
					t.Errorf("Expected database.ErrDuplicateKey")
				}
			}

			{
				err := database.RemoveESBRecord(context.TODO(), esbRecords[1].PoolNameIn)
				if err != nil {
					t.Error(err)
				}
				_, err = database.GetESBRecord(context.TODO(), esbRecords[1].PoolNameIn)
				if err != database.ErrNoSuchRecord {
					t.Error(err)
				}
				err = database.AddESBRecord(context.TODO(), esbRecords[1])
				if err != nil {
					t.Error(err)
				}
			}

			{
				esbRecord, err := database.GetESBRecord(context.TODO(), esbRecords[0].PoolNameIn)
				if err != nil {
					t.Error(err)
				}
				if esbRecord != esbRecords[0] {
					t.Errorf("res != expected: %s != %s", esbRecord, esbRecords[0])
				}
			}

			{
				for _, expectedESBRecord := range esbRecords {
					esbRecord, err := database.GetESBRecord(context.TODO(), expectedESBRecord.PoolNameIn)
					if err != nil {
						t.Error(err)
					}
					if esbRecord != expectedESBRecord {
						t.Errorf("res != expected: %s != %s", esbRecord, expectedESBRecord)
					}
				}
			}

		})
	}
}
