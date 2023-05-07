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

type staticEndpoints struct {
	coll  *mongo.Collection
	cache gcache.Cache
	mutex sync.RWMutex
}

func createStaticEndpoints(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) (*staticEndpoints, error) {
	se := &staticEndpoints{}
	err := se.init(ctx, client, cfg)
	return se, err
}

func (s *staticEndpoints) init(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) error {
	s.coll = client.Database(DATABASE_NAME).Collection(STATIC_ENDPOINTS_COLLECTION)
	s.cache = gcache.New(cfg.CacheSize).Simple().LoaderFunc(func(path interface{}) (interface{}, error) {
		var res StaticEndpoint
		err := s.coll.FindOne(
			ctx,
			bson.D{primitive.E{Key: STATIC_ENDPOINT_PATH_FIELD, Value: path.(string)}},
		).Decode(&res)
		return res, err
	}).Build()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: STATIC_ENDPOINT_PATH_FIELD, Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := s.coll.Indexes().CreateOne(ctx, indexModel)
	return err
}

func (s *staticEndpoints) addStaticEndpoint(ctx context.Context, staticEndpoint StaticEndpoint) error {
	return util.RunWithWriteLock(&s.mutex, func() error {
		_, err := s.coll.InsertOne(
			ctx,
			staticEndpoint,
		)
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicateKey
		} else if err != nil {
			return err
		}
		err = s.cache.Set(staticEndpoint.Path, staticEndpoint)
		return err
	})
}

func (s *staticEndpoints) removeStaticEndpoint(ctx context.Context, path string) error {
	return util.RunWithWriteLock(&s.mutex, func() error {
		_, err := s.coll.DeleteOne(
			ctx,
			bson.D{primitive.E{Key: STATIC_ENDPOINT_PATH_FIELD, Value: path}},
		)
		if err != nil {
			return err
		}
		s.cache.Remove(path)
		return nil
	})
}

func (s *staticEndpoints) updateStaticEndpoint(ctx context.Context, staticEndpoint StaticEndpoint) error {
	return util.RunWithWriteLock(&s.mutex, func() error {
		_, err := s.coll.UpdateOne(
			ctx,
			bson.D{primitive.E{Key: STATIC_ENDPOINT_PATH_FIELD, Value: staticEndpoint}},
			bson.D{{Key: "$set", Value: bson.D{{Key: STATIC_ENDPOINT_RESPONSE_FIELD, Value: staticEndpoint.Response}}}},
		)
		if err == mongo.ErrNoDocuments {
			return ErrNoSuchPath
		} else if err != nil {
			return err
		}
		err = s.cache.Set(staticEndpoint.Path, staticEndpoint)
		return err
	})
}

func (s *staticEndpoints) getStaticEndpointResponse(ctx context.Context, path string) (string, error) {
	return util.RunWithReadLock(&s.mutex, func() (string, error) {
		// if key doesn't exist in cache, it will be fetched via LoadFunc from database
		res, err := s.cache.Get(path)
		if err == mongo.ErrNoDocuments {
			return "", ErrNoSuchPath
		} else if err != nil {
			return "", err
		}
		return res.(StaticEndpoint).Response, nil
	})
}

func (s *staticEndpoints) listAllStaticEndpointPaths(ctx context.Context) ([]string, error) {
	return util.RunWithReadLock(&s.mutex, func() ([]string, error) {
		opts := options.Find()
		opts = opts.SetSort(bson.D{{Key: "timestamp", Value: 1}})
		opts = opts.SetProjection(bson.D{{Key: STATIC_ENDPOINT_PATH_FIELD, Value: 1}})
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
