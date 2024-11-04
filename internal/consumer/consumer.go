package consumer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"cloud.google.com/go/pubsub"
	"github.com/DisKiDKelp/mini-scan-takehome/internal/db"
	"github.com/DisKiDKelp/mini-scan-takehome/pkg/scanning"
)

func processMessage(ctx context.Context, database *db.DB, msg *pubsub.Message) error {
    var scan scanning.Scan
    if err := json.Unmarshal(msg.Data, &scan); err != nil {
        return fmt.Errorf("failed to unmarshal scan data: %w", err)
    }

    log.Printf("Processing scan for IP %s, Port %d, Service %s", scan.Ip, scan.Port, scan.Service)

	var response string
	switch scan.DataVersion {
        case scanning.V1:
            // Type assertion to handle map[string]interface{} for V1
            v1DataMap, ok := scan.Data.(map[string]interface{})
            if !ok {
                return fmt.Errorf("unexpected data type for V1 data")
            }

            // Extract "response_bytes_utf8" field, which should be a base64 string
            responseBytes, ok := v1DataMap["response_bytes_utf8"].(string)
            if !ok {
                return fmt.Errorf("failed to extract response_bytes_utf8 from V1 data")
            }

            // Decode base64 to get the actual response
            decodedResponse, err := base64.StdEncoding.DecodeString(responseBytes)
            if err != nil {
                return fmt.Errorf("failed to decode base64 response: %w", err)
            }

            response = string(decodedResponse)

        case scanning.V2:
            // Type assertion to handle map[string]interface{} for V2
            v2DataMap, ok := scan.Data.(map[string]interface{})
            if !ok {
                return fmt.Errorf("unexpected data type for V2 data")
            }

            // Extract "response_str" field directly as string
            response, ok = v2DataMap["response_str"].(string)
            if !ok {
                return fmt.Errorf("failed to extract response_str from V2 data")
            }

        default:
            return fmt.Errorf("unknown data version: %d", scan.DataVersion)
	}


    // Step 1: Ensure IP/port entry exists and get its id
    ipAddressID, err := database.GetOrInsertIPAddress(ctx, scan.Ip, int(scan.Port), scan.Service, scan.Timestamp)
    if err != nil {
        log.Printf("Failed to get or insert IP address: %v", err)
        msg.Nack()
        return err
    }

    // Step 2: Update IP data (last_updated and service) for existing IP
    err = database.UpdateIPData(ctx, scan.Ip, int(scan.Port), scan.Service, scan.Timestamp)
    if err != nil {
        log.Printf("Failed to update IP data: %v", err)
        msg.Nack()
        return err
    }

    // Step 3: Insert message into the messages table
    err = database.InsertMessage(ctx, ipAddressID, string(msg.Data), scan.Service, response, scan.Timestamp)
    if err != nil {
        log.Printf("Failed to insert message: %v", err)
        msg.Nack()
        return err
    }

    log.Printf("Processed and stored message for IP %s, Port %d", scan.Ip, scan.Port)

    // Acknowledge the message after processing is complete
    msg.Ack()

    return nil
}

// Start initializes the message consumption process
func Start(ctx context.Context, database *db.DB, sub *pubsub.Subscription) error {

	return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		if err := processMessage(ctx, database, msg); err != nil {
			log.Printf("Error processing message: %v", err)
		}
	})
}