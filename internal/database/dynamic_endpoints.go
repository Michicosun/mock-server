package database

import (
	"context"
	"errors"

	"github.com/bluele/gcache"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const dynamicEndpointsCollection string = "dynamic_endpoints"

type dynamicEndpoints struct {
	coll  *mongo.Collection
	cache gcache.Cache
}

func (s *dynamicEndpoints) init(ctx context.Context, client *mongo.Client) {
	s.coll = client.Database(databaseName).Collection(dynamicEndpointsCollection)
	s.cache = gcache.New(0).Simple().LoaderFunc(func(path interface{}) (interface{}, error) {
		var res DynamicEndpoint
		err := s.coll.FindOne(
			ctx,
			bson.D{primitive.E{Key: "path", Value: path.(string)}},
		).Decode(&res)
		return res, err
	}).Build()
}

func (s *dynamicEndpoints) addDynamicEndpoint(ctx context.Context, dynamicEndpoint DynamicEndpoint) error {
	err := s.cache.Set(dynamicEndpoint.Path, dynamicEndpoint)
	if err != nil {
		return err
	}
	_, err = s.coll.InsertOne(
		ctx,
		dynamicEndpoint,
	)
	return err
}

func (s *dynamicEndpoints) removeDynamicEndpoint(ctx context.Context, path string) error {
	s.cache.Remove(path)
	_, err := s.coll.DeleteOne(
		ctx,
		bson.D{primitive.E{Key: "path", Value: path}},
	)

	return err
}

func (s *dynamicEndpoints) getDynamicEndpointScriptName(ctx context.Context, path string) (string, error) {
	// if key doesn't exist in cache, it will be fetched via LoadFunc from database
	res, err := s.cache.Get(path)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", errors.New("no such path")
		} else {
			return "", err
		}
	}
	return res.(DynamicEndpoint).ScriptName, nil
}

func (s *dynamicEndpoints) listAllDynamicEndpointPaths(ctx context.Context) ([]string, error) {
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}, {Key: "_id", Value: 1}})
	opts = opts.SetSort(bson.D{{Key: "timestamp", Value: 1}, {Key: "_id", Value: 1}})
	opts = opts.SetProjection(bson.D{{Key: "path", Value: 1}})
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
