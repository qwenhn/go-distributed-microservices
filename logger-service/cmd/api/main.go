package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"

	"logger/data"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const webPort = 8080
const rpcPort = 5001
const gRPCPort = 50001

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

	rpcServer := &RPCServer{
		DB: db,
	}

	err := rpc.Register(rpcServer)
	if err != nil {
		log.Fatalln(err)
	}

	go app.rpcListen()

	go app.gRPCListen()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	log.Printf("Starting logger-service on port %d", webPort)

	err = server.ListenAndServe()
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

func (app *Application) rpcListen() error {
	log.Println("Listening RPC server on port :", rpcPort)
	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", rpcPort))
	if err != nil {
		return err
	}
	defer listen.Close()

	for {
		rpcConn, err := listen.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(rpcConn)
	}
}
