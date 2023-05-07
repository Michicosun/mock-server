package database

import (
	"context"
	"mock-server/internal/configs"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type taskMessages struct {
	coll *mongo.Collection
}

func createTaskMessages(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) (*taskMessages, error) {
	tm := &taskMessages{}
	err := tm.init(ctx, client, cfg)
	return tm, err
}

func (s *taskMessages) init(ctx context.Context, client *mongo.Client, cfg *configs.DatabaseConfig) error {
	s.coll = client.Database(DATABASE_NAME).Collection(TASK_MESSAGES_COLLECTION)
	return nil
}

func (s *taskMessages) addTaskMessage(ctx context.Context, taskMessage TaskMessage) error {
	_, err := s.coll.InsertOne(
		ctx,
		taskMessage,
	)
	if mongo.IsDuplicateKeyError(err) {
		return ErrDuplicateKey
	}
	return err
}

func (s *taskMessages) getTaskMessages(ctx context.Context, taskId string) ([]string, error) {
	opts := options.Find()
	opts = opts.SetSort(bson.D{{Key: "timestamp", Value: 1}})
	opts = opts.SetProjection(bson.D{{Key: MESSAGE_FIELD, Value: 1}})
	cursor, err := s.coll.Find(
		ctx,
		bson.D{{Key: TASK_ID_FIELD, Value: taskId}},
		opts,
	)
	if err != nil {
		return nil, err
	}

	var result []TaskMessage
	if err := cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	var messages = make([]string, len(result))
	for i, taskMessage := range result {
		messages[i] = taskMessage.Message
	}
	return messages, nil
}
