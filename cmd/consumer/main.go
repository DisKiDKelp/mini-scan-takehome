package main

import (
	"context"
	"flag"
	"log"

	"cloud.google.com/go/pubsub"
	"github.com/DisKiDKelp/mini-scan-takehome/internal/consumer"
	"github.com/DisKiDKelp/mini-scan-takehome/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()

	ctx := context.Background()

	projectId := flag.String("project", "test-project", "GCP Project ID")

	// Initialize SQLite database
	database, err := db.NewPostgresConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("Connected Database.")

	client, err := pubsub.NewClient(ctx, *projectId)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	log.Println("Connected Pub/Sub.")

	sub := client.Subscription("scan-sub")

	// Start the consumer
	if err := consumer.Start(ctx, database, sub); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}
}
