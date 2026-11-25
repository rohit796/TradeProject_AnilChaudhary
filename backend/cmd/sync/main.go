package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"trade-reporting-backend/database"
	"trade-reporting-backend/elastic"
	"trade-reporting-backend/kafka"
)

// One-shot synchronization utility:
// Reads all trades from MSSQL (LAPTOP-OV8LM74T\MSSQLSERVERANIL) -> Publishes to Kafka topic -> Indexes to Elasticsearch
func main() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("backend/.env")
	_ = godotenv.Load(`D:\TradeProject\backend\.env`)

	// Shared connection logic: SQL login when MSSQL_USER/MSSQL_PASSWORD are set
	// (e.g. the Docker container), otherwise Windows Authentication (trusted_connection).
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Failed to connect to MSSQL: %v", err)
	}
	defer db.Close()

	log.Printf("[Sync Utility] Connected to MSSQL (%s / %s)", os.Getenv("MSSQL_SERVER"), os.Getenv("MSSQL_DATABASE"))

	// Init Kafka & Elastic
	producer := kafka.InitProducer()
	defer producer.Close()

	esClient := elastic.InitElasticsearch()

	// Shared query (same joins and local-time handling as the API's /sync/trades)
	reports, err := database.GetAllTradesForSync()
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	count := 0
	for _, r := range reports {
		// 1. Stream the full enriched report to Kafka (a partial Trade would overwrite
		//    status/brokerage/taxes when the consumer indexes it)
		_ = producer.PublishTradeReport(context.Background(), &r)

		// 2. Index to Elasticsearch
		if esClient.IsReady() {
			r.IndexedAt = time.Now()
			_ = esClient.IndexTradeReport(context.Background(), &r)
		}

		count++
		log.Printf("[Sync Utility] Synced %s (%s %d shares @ ₹%.2f) to Kafka & Elastic", r.TradeID, r.TradeType, r.Quantity, r.Price)
	}

	log.Printf("[Sync Utility] Completed sync of %d trade records!", count)
}
