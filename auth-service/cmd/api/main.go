package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"auth/data"
)

const webPort = 8080

type Application struct {
	DB     *sql.DB
	Models data.Models
}

func main() {
	log.Println("Starting Auth Service")

	db := connectToDB()
	if db == nil {
		log.Fatal("Couldn't connect to Postgres")
		return
	}

	app := Application{
		DB:     db,
		Models: data.New(db),
	}

	log.Printf("Starting auth-service on port %d", webPort)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func connectToDB() *sql.DB {
	dsn := os.Getenv("DSN")

	for i := range 10 {
		db, err := sql.Open("pgx", dsn)

		if err == nil && db.Ping() == nil {
			log.Println("Connected to Postgres")
			return db
		}

		log.Printf("Waiting for Postgres (attempt %d/10)", i+1)
		time.Sleep(time.Second * 2)
	}

	log.Println("Failed to connect to Postgres after 10 attempts")
	return nil
}
