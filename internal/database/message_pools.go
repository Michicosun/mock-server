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

type messagePools struct {
	coll  *mongo.Collection
	cache gcache.Cache
	mutex sync.RWMutex
}

func createMessagePools(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) (*messagePools, error) {
	mp := &messagePools{}
	err := mp.init(ctx, client, cfg)
	return mp, err
}

func (mp *messagePools) init(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) error {
	mp.coll = client.Database(DATABASE_NAME).Collection(MESSAGE_POOLS_COLLECTION)
	mp.cache = gcache.New(cfg.CacheSize).Simple().LoaderFunc(func(poolName interface{}) (interface{}, error) {
		var res MessagePool
		err := mp.coll.FindOne(
			ctx,
			bson.D{primitive.E{Key: MESSAGE_POOL_NAME, Value: poolName.(string)}},
		).Decode(&res)
		return res, err
	}).Build()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: MESSAGE_POOL_NAME, Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := mp.coll.Indexes().CreateOne(ctx, indexModel)
	return err
}

func (mp *messagePools) addMessagePool(ctx context.Context, messagePool MessagePool) error {
	return util.RunWithWriteLock(&mp.mutex, func() error {
		_, err := mp.coll.InsertOne(
			ctx,
			messagePool,
		)
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicateKey
		} else if err != nil {
			return err
		}
		err = mp.cache.Set(messagePool.Name, messagePool)
		return err
	})
}

func (mp *messagePools) removeMessagePool(ctx context.Context, name string) error {
	return util.RunWithWriteLock(&mp.mutex, func() error {
		_, err := mp.coll.DeleteOne(
			ctx,
			bson.D{primitive.E{Key: MESSAGE_POOL_NAME, Value: name}},
		)
		if err == mongo.ErrNoDocuments {
			return ErrNoSuchPool
		}
		if err != nil {
			return err
		}
		mp.cache.Remove(name)
		return nil
	})
}

func (mp *messagePools) getMessagePool(ctx context.Context, name string) (MessagePool, error) {
	return util.RunWithReadLock(&mp.mutex, func() (MessagePool, error) {
		res, err := mp.cache.Get(name)
		if err == mongo.ErrNoDocuments {
			return MessagePool{}, ErrNoSuchPool
		} else if err != nil {
			return MessagePool{}, err
		}
		return res.(MessagePool), nil
	})
}
