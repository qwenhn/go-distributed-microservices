package data

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

var dbTimeout = 3 * time.Second

var client *mongo.Client

type Models struct {
	LogEntry LogEntry
}

func New(m *mongo.Client) Models {
	client = m

	return Models{
		LogEntry: LogEntry{},
	}
}
