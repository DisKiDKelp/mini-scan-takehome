package main

import (
	"context"
	"os"
	"flag"
	"fmt"
	"github.com/DisKiDKelp/mini-scan-takehome/internal/db"
	"github.com/DisKiDKelp/mini-scan-takehome/internal/consumer"
	"cloud.google.com/go/pubsub"
)

func main() {
	ctx := context.Background()

	projectId := flag.String("project", "test-project", "GCP Project ID")

	client, err := pubsub.NewClient(ctx, *projectId)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	sub := client.Subscription("scan-sub")

	// Initialize SQLite database
	database, err := db.NewSQLiteConnection("file:mydatabase.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatalf("Failed to connect to SQLite: %v", err)
	}
	defer database.Close()

	// Start the consumer
	if err := consumer.Start(ctx, database, client); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}
}