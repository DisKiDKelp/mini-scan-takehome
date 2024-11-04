package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"cloud.google.com/go/pubsub"
	"github.com/DisKiDKelp/mini-scan-takehome/internal/db"
	"github.com/DisKiDKelp/mini-scan-takehome/pkg/scanning"
)

// processMessage parses the message and updates the database
func processMessage(ctx context.Context, database *db.DB, msg *pubsub.Message) error {
	var scan scanning.Scan
	if err := json.Unmarshal(msg.Data, &scan); err != nil {
		return fmt.Errorf("failed to unmarshal scan data: %w", err)
	}

	var response string
	switch scan.DataVersion {
		case scanning.V1:
			v1Data := scanning.V1Data{}
			if err := json.Unmarshal(scan.Data.([]byte), &v1Data); err != nil {
				return fmt.Errorf("failed to unmarshal v1 data: %w", err)
			}
			response = string(v1Data.ResponseBytesUtf8)

		case scanning.V2:
			v2Data := scanning.V2Data{}
			if err := json.Unmarshal(scan.Data.([]byte), &v2Data); err != nil {
				return fmt.Errorf("failed to unmarshal v2 data: %w", err)
			}
			response = v2Data.ResponseStr

		default:
			return fmt.Errorf("unknown data version: %d", scan.DataVersion)
	}

	// Acquire lock before updating the database
	lockID, err := database.AcquireLock(ctx, scan.Ip, int(scan.Port))
	if err != nil {
		log.Printf("Failed to acquire lock: %v", err)
		msg.Nack()
		return err
	}
	defer database.ReleaseLock(ctx, scan.Ip, int(scan.Port), lockID)

	// Update the database with parsed information
	if err := database.UpdateIPData(ctx, scan.Ip, int(scan.Port), scan.Service, scan.Timestamp, response); err != nil {
		return fmt.Errorf("failed to update IP data: %w", err)
	}

	// Acknowledge the message only if processing is complete
	msg.Ack()
	return nil
}

// Start initializes the message consumption process
func Start(ctx context.Context, database *db.DB, client *pubsub.Client) error {
	return client.Receive(ctx, func(msg *pubsub.Message) {
		if err := processMessage(ctx, database, msg); err != nil {
			log.Printf("Error processing message: %v", err)
		}
	})
}