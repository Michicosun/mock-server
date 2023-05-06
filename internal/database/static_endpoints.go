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

type staticEndpoints struct {
	coll  *mongo.Collection
	cache gcache.Cache
	mutex sync.RWMutex
}

func createStaticEndpoints(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) *staticEndpoints {
	se := &staticEndpoints{}
	se.init(ctx, client, cfg)
	return se
}

func (s *staticEndpoints) init(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) {
	s.coll = client.Database(DATABASE_NAME).Collection(STATIC_ENDPOINTS_COLLECTION)
	s.cache = gcache.New(cfg.CacheSize).Simple().LoaderFunc(func(path interface{}) (interface{}, error) {
		var res StaticEndpoint
		err := s.coll.FindOne(
			ctx,
			bson.D{primitive.E{Key: STATIC_ENDPOINT_PATH, Value: path.(string)}},
		).Decode(&res)
		return res, err
	}).Build()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: STATIC_ENDPOINT_PATH, Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := s.coll.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		panic(err)
	}
}

func (s *staticEndpoints) addStaticEndpoint(ctx context.Context, staticEndpoint StaticEndpoint) error {
	return runWithWriteLock(&s.mutex, func() error {
		err := s.cache.Set(staticEndpoint.Path, staticEndpoint)
		if err != nil {
			return err
		}
		_, err = s.coll.InsertOne(
			ctx,
			staticEndpoint,
		)
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return err
	})
}

func (s *staticEndpoints) removeStaticEndpoint(ctx context.Context, path string) error {
	return runWithWriteLock(&s.mutex, func() error {
		s.cache.Remove(path)
		_, err := s.coll.DeleteOne(
			ctx,
			bson.D{primitive.E{Key: STATIC_ENDPOINT_PATH, Value: path}},
		)

		return err
	})
}

func (s *staticEndpoints) getStaticEndpointResponse(ctx context.Context, path string) (string, error) {
	return runWithReadLock(&s.mutex, func() (string, error) {
		// if key doesn't exist in cache, it will be fetched via LoadFunc from database
		res, err := s.cache.Get(path)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return "", ErrNoSuchPath
			} else {
				return "", err
			}
		}
		return res.(StaticEndpoint).Response, nil
	})
}

func (s *staticEndpoints) listAllStaticEndpointPaths(ctx context.Context) ([]string, error) {
	return runWithReadLock(&s.mutex, func() ([]string, error) {
		opts := options.Find()
		opts = opts.SetSort(bson.D{{Key: "timestamp", Value: 1}})
		opts = opts.SetProjection(bson.D{{Key: STATIC_ENDPOINT_PATH, Value: 1}})
		cursor, err := s.coll.Find(ctx, bson.D{}, opts)
		if err != nil {
			return nil, err
		}
		var results = []StaticEndpoint{}
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
