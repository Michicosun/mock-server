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

const staticEndpointsCollection string = "static_endpoints"

type staticEndpoints struct {
	coll  *mongo.Collection
	cache gcache.Cache
}

func (s *staticEndpoints) init(client *mongo.Client, ctx *context.Context) {
	s.coll = client.Database(databaseName).Collection(staticEndpointsCollection)
	s.cache = gcache.New(0).Simple().LoaderFunc(func(path interface{}) (interface{}, error) {
		var res StaticEndpoint
		err := s.coll.FindOne(
			*ctx,
			bson.D{primitive.E{Key: "path", Value: path.(string)}},
		).Decode(&res)
		return res, err
	}).Build()
}

func (s *staticEndpoints) addStaticEndpoint(staticEndpoint StaticEndpoint, ctx *context.Context) error {
	err := s.cache.Set(staticEndpoint.Path, staticEndpoint)
	if err != nil {
		return err
	}
	_, err = s.coll.InsertOne(
		*ctx,
		staticEndpoint,
	)
	return err
}

func (s *staticEndpoints) removeStaticEndpoint(path string, ctx *context.Context) error {
	s.cache.Remove(path)
	_, err := s.coll.DeleteOne(
		*ctx,
		bson.D{primitive.E{Key: "path", Value: path}},
	)

	return err
}

func (s *staticEndpoints) getStaticEndpointResponse(path string, ctx *context.Context) (string, error) {
	// if key doesn't exist in cache, it will be fetched via LoadFunc from database
	res, err := s.cache.Get(path)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", errors.New("no such path")
		} else {
			return "", err
		}
	}
	return res.(StaticEndpoint).Response, nil
}

func (s *staticEndpoints) listAllStaticEndpoints(ctx *context.Context) ([]StaticEndpoint, error) {
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}, {Key: "_id", Value: 1}})
	cursor, err := s.coll.Find(*ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	var results = []StaticEndpoint{}
	if err = cursor.All(*ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
