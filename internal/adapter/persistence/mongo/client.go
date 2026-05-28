package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connect opens a MongoDB connection and returns the selected database handle.
func Connect(ctx context.Context, host string, port int, user, pass, db string) (*mongo.Database, error) {
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d", user, pass, host, port)
	if user == "" {
		uri = fmt.Sprintf("mongodb://%s:%d", host, port)
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(cctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(cctx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}
	return client.Database(db), nil
}
