package database

import (
	"context"
	"mock-server/internal/configs"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func initInMemoryDB(ctx *context.Context) {
	testServer, err := mim.Start(context.Background(), "5.0.2")
	if err != nil {
		panic(err)
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(testServer.URI()))
	if err != nil {
		panic(err)
	}

	db.Init(client, ctx)
}

func InitDB(cfg *configs.DatabaseConfig, ctx *context.Context) {
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
}
