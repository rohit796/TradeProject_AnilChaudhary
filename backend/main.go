package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"trade-reporting-backend/database"
	"trade-reporting-backend/elastic"
	"trade-reporting-backend/handlers"
	"trade-reporting-backend/kafka"
)

// CORSMiddleware sets standard CORS headers for React frontend
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	// 1. Load environment variables (supports running from any terminal directory)
	if err := godotenv.Load(".env"); err != nil {
		if err2 := godotenv.Load("backend/.env"); err2 != nil {
			_ = godotenv.Load(`D:\TradeProject\backend\.env`)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("==================================================================")
	log.Println("  TradeFlow - Trade Reporting Pipeline & Backend Service")
	log.Println("  Architecture: SQL Server -> Kafka -> Elasticsearch -> Go -> React")
	log.Println("==================================================================")

	// 2. Initialize Microsoft SQL Server Connection
	if db, err := database.InitDB(); err != nil {
		log.Printf("[MSSQL] Warning: Could not connect to SQL Server: %v", err)
	} else {
		log.Printf("[MSSQL] Connection verified successfully (DB: %v)", db != nil)
	}

	// 3. Initialize Elasticsearch Client (Primary Data Source for Reports)
	esClient := elastic.InitElasticsearch()

	// 4. Initialize Kafka Producer
	producer := kafka.InitProducer()
	if producer != nil {
		defer producer.Close()
	}

	// 5. Start Kafka Background Consumer (Consumes from Kafka topic -> Indexes to Elastic)
	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()
	_ = kafka.StartConsumer(consumerCtx, esClient)

	// 5.1 Multi-Table Real-Time CDC: SQL Server -> Kafka -> Elasticsearch
	// Tracks changes on all 3 tables: trade_details (trade_id), client_details (client_id), stock_master (symbol)
	go func() {
		// Initialize high-watermarks to the CURRENT MAX timestamp in each table.
		// Zero old records are re-produced on startup. ONLY new updates/inserts will be streamed to Kafka.
		tradeWatermark, _ := database.GetMaxTradeUpdatedOn()
		clientWatermark, _ := database.GetMaxClientUpdatedOn()
		stockWatermark, _ := database.GetMaxStockUpdatedOn()

		log.Println("==================================================================")
		log.Println("[Incremental CDC] Multi-Table CDC engine active (Delta Changes Only):")
		log.Printf("  1. client_details high-watermark : %s", clientWatermark.Format("2006-01-02 15:04:05.000"))
		log.Printf("  2. stock_master   high-watermark : %s", stockWatermark.Format("2006-01-02 15:04:05.000"))
		log.Printf("  3. trade_details  high-watermark : %s", tradeWatermark.Format("2006-01-02 15:04:05.000"))
		log.Println("  Ready. Only modified/inserted rows will be produced to Kafka & Elasticsearch.")
		log.Println("==================================================================")

		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if producer == nil || esClient == nil || !esClient.IsReady() {
				continue
			}

			// 1. Check client_details for updates
			updatedClients, newClientWatermark, err := database.GetUpdatedClients(clientWatermark)
			if err == nil && len(updatedClients) > 0 {
				for _, c := range updatedClients {
					_ = producer.PublishClient(context.Background(), &c)
					_, _ = esClient.UpdateClientInTradeReports(context.Background(), &c)
					rowJSON, _ := json.Marshal(c)
					log.Println(string(rowJSON))
				}
				clientWatermark = newClientWatermark
			}

			// 2. Check stock_master for updates
			updatedStocks, newStockWatermark, err := database.GetUpdatedStocks(stockWatermark)
			if err == nil && len(updatedStocks) > 0 {
				for _, s := range updatedStocks {
					_ = producer.PublishStock(context.Background(), &s)
					_, _ = esClient.UpdateStockInTradeReports(context.Background(), &s)
					rowJSON, _ := json.Marshal(s)
					log.Println(string(rowJSON))
				}
				stockWatermark = newStockWatermark
			}

			// 3. Check trade_details for new or updated trades
			newTrades, newTradeWatermark, err := database.GetIncrementalTrades(tradeWatermark)
			if err == nil && len(newTrades) > 0 {
				for _, t := range newTrades {
					_ = producer.PublishTradeReport(context.Background(), &t)
					t.IndexedAt = time.Now()
					_ = esClient.IndexTradeReport(context.Background(), &t)
					rowJSON, _ := json.Marshal(t)
					log.Println(string(rowJSON))
				}
				tradeWatermark = newTradeWatermark
			}
		}
	}()

	// 6. Setup Gin HTTP Server with CORS & Silent Filter for /v1/models probe
	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/v1/models"},
	}))
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	// Silently answer if local IDE extensions probe /v1/models (never logged to terminal)
	router.GET("/v1/models", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"object": "list",
			"data":   []gin.H{},
		})
	})

	// 7. Register API Routes
	api := router.Group("/api")
	{
		api.GET("/health", handlers.HealthHandler)
		api.POST("/login", handlers.LoginHandler)

		// Master data from MSSQL
		api.GET("/clients", handlers.GetClientsHandler)
		api.GET("/stocks", handlers.GetStocksHandler)

		// Trade Execution (writes to MSSQL, streams to Kafka)
		api.POST("/trades", handlers.CreateTradeHandler)

		// Full Pipeline Sync: SQL Server -> Kafka topic -> Elasticsearch index
		api.POST("/sync/trades", handlers.SyncAllTradesFromSQLHandler)

		// Trade Reports (FETCHES STRICTLY FROM ELASTICSEARCH)
		api.GET("/reports/trades", handlers.GetTradeReportsHandler)

		// Kafka Topic Management Endpoints (REST for Postman)
		api.GET("/kafka/topics", handlers.ListKafkaTopicsHandler)
		api.POST("/kafka/topics", handlers.CreateKafkaTopicHandler)
		api.DELETE("/kafka/topics/:topic", handlers.DeleteKafkaTopicHandler)
	}

	log.Printf("[Server] Starting Go HTTP API on port %s (http://localhost:%s)", port, port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("[Server] Fatal: Failed to run server: %v", err)
	}
}
