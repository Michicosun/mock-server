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

type taskMessages struct {
	coll  *mongo.Collection
	cache gcache.Cache
	mutex sync.RWMutex
}

func createTaskMessages(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) (*taskMessages, error) {
	tm := &taskMessages{}
	err := tm.init(ctx, client, cfg)
	return tm, err
}

func (s *taskMessages) init(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) error {
	s.coll = client.Database(DATABASE_NAME).Collection(TASK_MESSAGES_COLLECTION)
	s.cache = gcache.New(cfg.CacheSize).Simple().LoaderFunc(func(task_id interface{}) (interface{}, error) {
		var res TaskMessage
		err := s.coll.FindOne(
			ctx,
			bson.D{primitive.E{Key: TASK_ID_FIELD, Value: task_id.(string)}},
		).Decode(&res)
		return res, err
	}).Build()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: TASK_ID_FIELD, Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := s.coll.Indexes().CreateOne(ctx, indexModel)
	return err
}

func (s *taskMessages) addTaskMessages(ctx context.Context, taskMessages TaskMessage) error {
	return util.RunWithWriteLock(&s.mutex, func() error {
		_, err := s.coll.InsertOne(
			ctx,
			taskMessages,
		)
		if mongo.IsDuplicateKeyError(err) {
			return nil
		} else if err != nil {
			return err
		}
		err = s.cache.Set(taskMessages.TaskId, taskMessages)
		return err
	})
}
