package database

import (
	"context"
	"errors"

	"github.com/bluele/gcache"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const staticEndpointsCollection string = "static_endpoints"

type staticEndpoints struct {
	coll  *mongo.Collection
	cache gcache.Cache
}

func (s *staticEndpoints) Init(client *mongo.Client) {
	s.coll = client.Database(databaseName).Collection(staticEndpointsCollection)
	s.cache = gcache.New(0).Simple().LoaderFunc(func(path interface{}) (interface{}, error) {
		var res StaticEndpoint
		err := s.coll.FindOne(
			context.TODO(),
			bson.D{primitive.E{Key: "path", Value: path.(string)}},
		).Decode(&res)
		return res, err
	}).Build()
}

func (s *staticEndpoints) addStaticEndpoint(staticEndpoint StaticEndpoint) error {
	s.cache.Set(staticEndpoint.Path, staticEndpoint)
	_, err := s.coll.InsertOne(
		context.TODO(),
		staticEndpoint,
	)
	return err
}

func (s *staticEndpoints) removeStaticEndpoint(path string) error {
	s.cache.Remove(path)
	_, err := s.coll.DeleteOne(
		context.TODO(),
		bson.D{primitive.E{Key: "path", Value: path}},
	)

	return err
}

func (s *staticEndpoints) getStaticEndpointResponse(path string) (string, error) {
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

func (s *staticEndpoints) listAllStaticEndpoints() ([]StaticEndpoint, error) {
	cursor, err := s.coll.Find(context.TODO(), bson.D{})
	if err != nil {
		return nil, err
	}
	var results = []StaticEndpoint{}
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (s *staticEndpoints) hasStaticEndpoint(path string) (bool, error) {
	if s.cache.Has(path) {
		return true, nil
	}
	if _, err := s.getStaticEndpointResponse(path); err != nil {
		if err == gcache.KeyNotFoundError {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}
