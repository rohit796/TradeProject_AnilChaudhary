package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/microsoft/go-mssqldb"
	"trade-reporting-backend/elastic"
	"trade-reporting-backend/kafka"
	"trade-reporting-backend/models"
)

// One-shot synchronization utility:
// Reads all trades from MSSQL (LAPTOP-OV8LM74T\MSSQLSERVERANIL) -> Publishes to Kafka topic -> Indexes to Elasticsearch
func main() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("backend/.env")
	_ = godotenv.Load(`D:\TradeProject\backend\.env`)

	server := os.Getenv("MSSQL_SERVER")
	if server == "" {
		server = `LAPTOP-OV8LM74T\MSSQLSERVERANIL`
	}
	dbName := os.Getenv("MSSQL_DATABASE")
	if dbName == "" {
		dbName = "TradeDB"
	}

	connStr := fmt.Sprintf("server=%s;database=%s;trusted_connection=yes;encrypt=disable;", server, dbName)
	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to MSSQL: %v", err)
	}
	defer db.Close()

	log.Printf("[Sync Utility] Connected to MSSQL (%s / %s)", server, dbName)

	// Init Kafka & Elastic
	producer := kafka.InitProducer()
	defer producer.Close()

	esClient := elastic.InitElasticsearch()

	query := `
		SELECT 
			t.trade_id, t.order_id, t.client_id, c.client_code, c.first_name, c.last_name, c.account_type,
			t.symbol, s.company_name, s.sector, t.exchange, t.segment, t.trade_type,
			t.quantity, t.price, t.trade_value, t.brokerage, t.taxes_charges, t.net_amount,
			t.trade_status, t.trade_time
		FROM trade_details t
		INNER JOIN client_details c ON t.client_id = c.client_id
		INNER JOIN stock_master s ON t.symbol = s.symbol
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var r models.TradeReport
		if err := rows.Scan(
			&r.TradeID, &r.OrderID, &r.ClientID, &r.ClientCode, &r.FirstName, &r.LastName, &r.AccountType,
			&r.Symbol, &r.CompanyName, &r.Sector, &r.Exchange, &r.Segment, &r.TradeType,
			&r.Quantity, &r.Price, &r.TradeValue, &r.Brokerage, &r.TaxesCharges, &r.NetAmount,
			&r.TradeStatus, &r.TradeTime,
		); err != nil {
			continue
		}

		trade := models.Trade{
			TradeID:    r.TradeID,
			OrderID:    r.OrderID,
			ClientID:   r.ClientID,
			Symbol:     r.Symbol,
			Exchange:   r.Exchange,
			Segment:    r.Segment,
			TradeType:  r.TradeType,
			Quantity:   r.Quantity,
			Price:      r.Price,
			TradeValue: r.TradeValue,
			NetAmount:  r.NetAmount,
			TradeTime:  r.TradeTime,
		}

		// 1. Stream to Kafka
		_ = producer.PublishTrade(context.Background(), &trade)

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
