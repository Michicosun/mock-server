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

type routes struct {
	coll  *mongo.Collection
	cache gcache.Cache
	mutex sync.RWMutex
}

func createRoutes(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) (*routes, error) {
	r := &routes{}
	err := r.init(ctx, client, cfg)
	return r, err
}

func (r *routes) init(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) error {
	r.coll = client.Database(DATABASE_NAME).Collection(ROUTES_COLLECTION)
	r.cache = gcache.New(cfg.CacheSize).Simple().LoaderFunc(func(path interface{}) (interface{}, error) {
		var res Route
		err := r.coll.FindOne(
			ctx,
			bson.D{primitive.E{Key: ROUTE_PATH_FIELD, Value: path.(string)}},
		).Decode(&res)
		return res, err
	}).Build()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: ROUTE_PATH_FIELD, Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := r.coll.Indexes().CreateOne(ctx, indexModel)
	return err
}

func (r *routes) addRoute(ctx context.Context, route Route) error {
	return util.RunWithWriteLock(&r.mutex, func() error {
		_, err := r.coll.InsertOne(
			ctx,
			route,
		)
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicateKey
		} else if err != nil {
			return err
		}
		err = r.cache.Set(route.Path, route)
		return err
	})
}

func (r *routes) removeRoute(ctx context.Context, path string) error {
	return util.RunWithWriteLock(&r.mutex, func() error {
		_, err := r.coll.DeleteOne(
			ctx,
			bson.D{primitive.E{Key: ROUTE_PATH_FIELD, Value: path}},
		)
		if err != nil {
			return err
		}
		r.cache.Remove(path)
		return nil
	})
}

func (s *routes) updateRoute(ctx context.Context, route Route) error {
	return util.RunWithWriteLock(&s.mutex, func() error {
		res, err := s.coll.UpdateOne(
			ctx,
			bson.D{{Key: "$and", Value: bson.A{
				bson.D{{Key: ROUTE_PATH_FIELD, Value: route.Path}},
				bson.D{{Key: ROUTE_TYPE_FIELD, Value: route.Type}}},
			}},
			bson.D{{Key: "$set", Value: bson.D{
				{Key: ROUTE_RESPONSE_FIELD, Value: route.Response},
				{Key: ROUTE_SCRIPT_NAME_FIELD, Value: route.ScriptName},
			}}},
		)
		if err == mongo.ErrNoDocuments || res.MatchedCount == 0 {
			return ErrNoSuchPath
		} else if err != nil {
			return err
		}
		err = s.cache.Set(route.Path, route)
		return err
	})
}

func (s *routes) getRoute(ctx context.Context, path string) (Route, error) {
	return util.RunWithReadLock(&s.mutex, func() (Route, error) {
		// if key doesn't exist in cache, it will be fetched via LoadFunc from database
		res, err := s.cache.Get(path)
		if err == mongo.ErrNoDocuments {
			return Route{}, ErrNoSuchPath
		} else if err != nil {
			return Route{}, err
		}
		return res.(Route), nil
	})
}

func (r *routes) listAllRoutesPathsWithType(ctx context.Context, t string) ([]string, error) {
	return util.RunWithReadLock(&r.mutex, func() ([]string, error) {
		opts := options.Find()
		opts = opts.SetSort(bson.D{{Key: "timestamp", Value: 1}})
		opts = opts.SetProjection(bson.D{{Key: ROUTE_PATH_FIELD, Value: 1}})
		cursor, err := r.coll.Find(ctx, bson.D{{Key: ROUTE_TYPE_FIELD, Value: t}}, opts)
		if err != nil {
			return nil, err
		}
		var results = []Route{}
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
