package database

import (
	"context"
	"mock-server/internal/configs"
	"mock-server/internal/util"
	"sync"

	"github.com/bluele/gcache"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type dynamicEndpoints struct {
	coll  *mongo.Collection
	cache gcache.Cache
	mutex sync.RWMutex
}

func createDynamicEndpoints(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) (*dynamicEndpoints, error) {
	de := &dynamicEndpoints{}
	err := de.init(ctx, client, cfg)
	return de, err
}

func (s *dynamicEndpoints) init(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) error {
	s.coll = client.Database(DATABASE_NAME).Collection(DYNAMIC_ENDPOINTS_COLLECTION)
	s.cache = gcache.New(cfg.CacheSize).Simple().LoaderFunc(func(path interface{}) (interface{}, error) {
		var res DynamicEndpoint
		err := s.coll.FindOne(
			ctx,
			bson.D{primitive.E{Key: DYNAMIC_ENDPOINT_PATH_FIELD, Value: path.(string)}},
		).Decode(&res)
		return res, err
	}).Build()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: DYNAMIC_ENDPOINT_PATH_FIELD, Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := s.coll.Indexes().CreateOne(ctx, indexModel)
	return err
}

func (s *dynamicEndpoints) addDynamicEndpoint(ctx context.Context, dynamicEndpoint DynamicEndpoint) error {
	return util.RunWithWriteLock(&s.mutex, func() error {
		_, err := s.coll.InsertOne(
			ctx,
			dynamicEndpoint,
		)
		if mongo.IsDuplicateKeyError(err) {
			return nil
		} else if err != nil {
			return err
		}
		err = s.cache.Set(dynamicEndpoint.Path, dynamicEndpoint)
		return err
	})
}

func (s *dynamicEndpoints) removeDynamicEndpoint(ctx context.Context, path string) error {
	return util.RunWithWriteLock(&s.mutex, func() error {
		_, err := s.coll.DeleteOne(
			ctx,
			bson.D{primitive.E{Key: DYNAMIC_ENDPOINT_PATH_FIELD, Value: path}},
		)
		if err != nil {
			return err
		}
		s.cache.Remove(path)
		return nil
	})
}

func (s *dynamicEndpoints) updateDynamicEndpoint(ctx context.Context, dynamicEndpoint DynamicEndpoint) error {
	return util.RunWithWriteLock(&s.mutex, func() error {
		_, err := s.coll.UpdateOne(
			ctx,
			bson.D{{Key: DYNAMIC_ENDPOINT_PATH_FIELD, Value: dynamicEndpoint}},
			bson.D{{Key: "$set", Value: bson.D{{Key: DYNAMIC_ENDPOINT_SCRIPT_NAME_FIELD, Value: dynamicEndpoint.ScriptName}}}},
		)
		if err == mongo.ErrNoDocuments {
			return ErrNoSuchPath
		} else if err != nil {
			return err
		}
		err = s.cache.Set(dynamicEndpoint.Path, dynamicEndpoint)
		return err
	})
}

func (s *dynamicEndpoints) getDynamicEndpointScriptName(ctx context.Context, path string) (string, error) {
	return util.RunWithReadLock(&s.mutex, func() (string, error) {
		// if key doesn't exist in cache, it will be fetched via LoadFunc from database
		res, err := s.cache.Get(path)
		if err == mongo.ErrNoDocuments {
			return "", ErrNoSuchPath
		} else if err != nil {
			return "", err
		}
		return res.(DynamicEndpoint).ScriptName, nil
	})
}

func (s *dynamicEndpoints) listAllDynamicEndpointPaths(ctx context.Context) ([]string, error) {
	return util.RunWithReadLock(&s.mutex, func() ([]string, error) {
		opts := options.Find()
		opts = opts.SetSort(bson.D{{Key: "timestamp", Value: 1}})
		opts = opts.SetProjection(bson.D{{Key: DYNAMIC_ENDPOINT_PATH_FIELD, Value: 1}})
		cursor, err := s.coll.Find(ctx, bson.D{}, opts)
		if err != nil {
			return nil, err
		}
		var results = []DynamicEndpoint{}
		if err = cursor.All(ctx, &results); err != nil {
			return nil, err
		}
		paths := make([]string, len(results))
		for i := 0; i < len(results); i++ {
			paths[i] = results[i].Path
		}
		return paths, nil
	})
}
