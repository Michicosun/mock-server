package database

import (
	"context"
	"mock-server/internal/configs"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var inMemoryServer mim.Server

func initInMemoryDB(ctx context.Context, cfg *configs.DatabaseConfig) error {
	inMemoryServer, err := mim.Start(ctx, "5.0.2")
	if err != nil {
		return err
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(inMemoryServer.URI()))
	if err != nil {
		return err
	}

	return db.init(ctx, client, cfg)
}

func InitDB(ctx context.Context, cfg *configs.DatabaseConfig) error {
	if cfg.InMemory {
		return initInMemoryDB(ctx, cfg)
	} else {
		panic("Not implemented")
	}
}

func Disconnect(ctx context.Context) {
	err := db.client.Disconnect(ctx)
	if err != nil {
		panic(err)
	}
	inMemoryServer.Stop(ctx)
}
