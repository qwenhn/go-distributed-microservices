package main

import (
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"listener/lib/event"
)

func main() {
	log.Println("Starting Listener Service")

	// #1 try to connect to RabbitMQ (with re-try)
	conn, err := connectToRabbitMQ()
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	// #2 start listening to messages
	log.Println("Listening and consuming for RabbitMQ messages ...")

	// create consumer
	consumer, err := event.NewConsumer(conn)
	if err != nil {
		panic(err)
	}

	err = consumer.Listen([]string{"log.INFO", "log.WARNING", "log.ERROR"})
	if err != nil {
		log.Println(err)
		return
	}
}

func connectToRabbitMQ() (*amqp.Connection, error) {
	var conn *amqp.Connection
	var err error

	maxRetries := 10

	for i := 1; i <= maxRetries; i++ {
		conn, err = amqp.Dial(os.Getenv("RABBITMQ_URL"))
		if err == nil {
			log.Println("Connected to RabbitMQ")
			return conn, nil
		}

		log.Printf("RabbitMQ not ready yet (attempt %d/%d): %v", i, maxRetries, err)

		backOff := time.Duration(i*i) * time.Second
		log.Printf("Backing off for %v...", backOff)

		time.Sleep(backOff)
	}

	return nil, err
}
