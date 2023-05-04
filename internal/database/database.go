package database

import (
	"context"
	"mock-server/internal/configs"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var inMemoryServer mim.Server

func initInMemoryDB(ctx context.Context) {
	inMemoryServer, err := mim.Start(ctx, "5.0.2")
	if err != nil {
		panic(err)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(inMemoryServer.URI()))
	if err != nil {
		panic(err)
	}

	db.init(ctx, client)
}

func InitDB(cfg *configs.DatabaseConfig, ctx context.Context) {
	if cfg.InMemory {
		initInMemoryDB(ctx)
	} else {
		panic("Not implemented")
	}
}

func Disconnect() {
	err := db.client.Disconnect(context.Background())
	if err != nil {
		panic(err)
	}
	inMemoryServer.Stop(context.Background())
}
