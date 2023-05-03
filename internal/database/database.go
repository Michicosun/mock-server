package database

import (
	"context"
	"mock-server/internal/configs"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func initInMemoryDB() {
	testServer, err := mim.Start(context.Background(), "5.0.2")
	if err != nil {
		panic(err)
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(testServer.URI()))
	if err != nil {
		panic(err)
	}

	db = &MongoStorage{}
	db.Init(client)
}

func InitDB(cfg *configs.DatabaseConfig) {
	if cfg.InMemory {
		initInMemoryDB()
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
