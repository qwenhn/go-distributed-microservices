package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"logger/data"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const webPort = 8080

type Application struct {
	DB     *mongo.Client
	Models data.Models
}

func main() {
	log.Println("Starting Logger Service")

	db := connectToDB()
	if db == nil {
		log.Fatal("Couldn't connect to Mongo")
		return
	}

	app := Application{
		DB:     db,
		Models: data.New(db),
	}

	log.Printf("Starting logger-service on port %d", webPort)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func connectToDB() *mongo.Client {
	uri := os.Getenv("MONGO_URI")

	for range 10 {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		client, err := mongo.Connect(options.Client().ApplyURI(uri))
		if err == nil {
			err = client.Ping(ctx, nil)
			cancel()

			if err == nil {
				log.Println("Connected to Mongo")
				return client
			}
		}

		cancel()

		log.Println("Waiting for Mongo")
		time.Sleep(2 * time.Second)
	}

	return nil
}
