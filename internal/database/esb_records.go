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

type esbRecords struct {
	coll  *mongo.Collection
	cache gcache.Cache
	mutex sync.RWMutex
}

func createESBRecords(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) (*esbRecords, error) {
	esb := &esbRecords{}
	err := esb.init(ctx, client, cfg)
	return esb, err
}

func (esb *esbRecords) init(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) error {
	esb.coll = client.Database(DATABASE_NAME).Collection(ESB_RECORDS_COLLECTION)
	esb.cache = gcache.New(cfg.CacheSize).Simple().LoaderFunc(func(poolNameIn interface{}) (interface{}, error) {
		var res StaticEndpoint
		err := esb.coll.FindOne(
			ctx,
			bson.D{primitive.E{Key: POOL_NAME_IN_FIELD, Value: poolNameIn.(string)}},
		).Decode(&res)
		return res, err
	}).Build()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: POOL_NAME_IN_FIELD, Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := esb.coll.Indexes().CreateOne(ctx, indexModel)
	return err
}

func (esb *esbRecords) addESBRecord(ctx context.Context, esbRecord ESBRecord) error {
	return util.RunWithWriteLock(&esb.mutex, func() error {
		_, err := esb.coll.InsertOne(
			ctx,
			esbRecord,
		)
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicateKey
		} else if err != nil {
			return err
		}
		err = esb.cache.Set(esbRecord.PoolNameIn, esbRecord)
		return err
	})
}

func (esb *esbRecords) removeESBRecord(ctx context.Context, poolNameIn string) error {
	return util.RunWithWriteLock(&esb.mutex, func() error {
		_, err := esb.coll.DeleteOne(
			ctx,
			bson.D{primitive.E{Key: POOL_NAME_IN_FIELD, Value: poolNameIn}},
		)
		if err != nil {
			return err
		}
		esb.cache.Remove(poolNameIn)
		return nil
	})
}

func (esb *esbRecords) getESBRecord(ctx context.Context, poolNameIn string) (ESBRecord, error) {
	return util.RunWithReadLock(&esb.mutex, func() (ESBRecord, error) {
		var res ESBRecord
		err := esb.coll.FindOne(
			ctx,
			bson.D{{Key: POOL_NAME_IN_FIELD, Value: poolNameIn}},
			nil,
		).Decode(&res)

		if err == mongo.ErrNoDocuments {
			return res, ErrNoSuchRecord
		}
		return res, err
	})
}
