package database

import (
	"context"
	"mock-server/internal/configs"
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
	mutex sync.Mutex
}

func createDynamicEndpoints(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) *dynamicEndpoints {
	de := &dynamicEndpoints{}
	de.init(ctx, client, cfg)
	return de
}

func (s *dynamicEndpoints) init(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) {
	s.coll = client.Database(DATABASE_NAME).Collection(DYNAMIC_ENDPOINTS_COLLECTION)
	s.cache = gcache.New(cfg.CacheSize).Simple().LoaderFunc(func(path interface{}) (interface{}, error) {
		var res DynamicEndpoint
		err := s.coll.FindOne(
			ctx,
			bson.D{primitive.E{Key: DYNAMIC_ENDPOINT_PATH, Value: path.(string)}},
		).Decode(&res)
		return res, err
	}).Build()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: DYNAMIC_ENDPOINT_PATH, Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := s.coll.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		panic(err)
	}
}

func (s *dynamicEndpoints) addDynamicEndpoint(ctx context.Context, dynamicEndpoint DynamicEndpoint) error {
	return runWithLock(&s.mutex, func() error {
		err := s.cache.Set(dynamicEndpoint.Path, dynamicEndpoint)
		if err != nil {
			return err
		}
		_, err = s.coll.InsertOne(
			ctx,
			dynamicEndpoint,
		)
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return err
	})
}

func (s *dynamicEndpoints) removeDynamicEndpoint(ctx context.Context, path string) error {
	return runWithLock(&s.mutex, func() error {
		s.cache.Remove(path)
		_, err := s.coll.DeleteOne(
			ctx,
			bson.D{primitive.E{Key: DYNAMIC_ENDPOINT_PATH, Value: path}},
		)
		return err
	})
}

func (s *dynamicEndpoints) getDynamicEndpointScriptName(ctx context.Context, path string) (string, error) {
	// if key doesn't exist in cache, it will be fetched via LoadFunc from database
	res, err := s.cache.Get(path)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", ErrNoSuchPath
		} else {
			return "", err
		}
	}
	return res.(DynamicEndpoint).ScriptName, nil
}

func (s *dynamicEndpoints) listAllDynamicEndpointPaths(ctx context.Context) ([]string, error) {
	opts := options.Find()
	opts = opts.SetSort(bson.D{{Key: "timestamp", Value: 1}})
	opts = opts.SetProjection(bson.D{{Key: DYNAMIC_ENDPOINT_PATH, Value: 1}})
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
}
