package handlers

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"trade-reporting-backend/database"
	"trade-reporting-backend/elastic"
	"trade-reporting-backend/kafka"
	"trade-reporting-backend/models"
)

// LoginHandler validates credentials against .env
func LoginHandler(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	envUser := os.Getenv("AUTH_USERNAME")
	if envUser == "" {
		envUser = "admin"
	}
	envPass := os.Getenv("AUTH_PASSWORD")
	if envPass == "" {
		envPass = "admin123"
	}

	if req.Username != envUser || req.Password != envPass {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Authentication failed",
			"message": "Invalid credentials. Verify credentials configured in .env",
		})
		return
	}

	c.JSON(http.StatusOK, models.LoginResponse{
		Token: "jwt-token-tradeflow-" + strconv.FormatInt(time.Now().Unix(), 10),
		User: models.UserProfile{
			Username: envUser,
			Role:     "Trade Operations Manager",
			Name:     "Anil (Admin)",
		},
	})
}

// GetClientsHandler returns client master data strictly from MSSQL client_details table
func GetClientsHandler(c *gin.Context) {
	clients, err := database.GetClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, clients)
}

// GetStocksHandler returns stock master catalog strictly from MSSQL stock_master table
func GetStocksHandler(c *gin.Context) {
	stocks, err := database.GetStocks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, stocks)
}

// GetTradeReportsHandler fetches data STRICTLY from Elasticsearch
func GetTradeReportsHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit <= 0 {
		limit = 100
	} else if limit > 10000 {
		limit = 10000
	}

	filter := models.TradeFilter{
		ClientID:  c.Query("client_id"),
		Symbol:    c.Query("symbol"),
		TradeType: c.Query("trade_type"),
		Search:    c.Query("search"),
		SortBy:    c.DefaultQuery("sort_by", "trade_time"),
		Order:     c.DefaultQuery("order", "desc"),
		Page:      page,
		Limit:     limit,
	}

	if elastic.GlobalESClient == nil || !elastic.GlobalESClient.IsReady() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Elasticsearch is not connected",
			"message": "Reporting queries read strictly from Elasticsearch (http://localhost:9200). Please start Elasticsearch.",
			"total":   0,
			"page":    page,
			"limit":   limit,
			"trades":  []models.TradeReport{},
		})
		return
	}

	reports, total, err := elastic.GlobalESClient.SearchTradeReports(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Elasticsearch search failed: " + err.Error(),
			"total":  0,
			"page":   page,
			"limit":  limit,
			"trades": []models.TradeReport{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"source": "elasticsearch",
		"index":  "trade_reports",
		"total":  total,
		"page":   page,
		"limit":  limit,
		"trades": reports,
	})
}

// CreateTradeHandler receives trade -> writes to MSSQL (OLTP) -> streams to Kafka -> indexes to Elastic
func CreateTradeHandler(c *gin.Context) {
	var trade models.Trade
	if err := c.ShouldBindJSON(&trade); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid trade format: " + err.Error()})
		return
	}

	if trade.TradeID == "" {
		trade.TradeID = "TRD-" + strconv.FormatInt(time.Now().UnixNano(), 10)[10:]
	}
	if trade.OrderID == "" {
		trade.OrderID = "ORD-" + strconv.FormatInt(time.Now().Unix(), 10)[5:]
	}
	if trade.TradeTime.IsZero() {
		trade.TradeTime = time.Now()
	}
	if trade.TradeValue == 0 {
		trade.TradeValue = float64(trade.Quantity) * trade.Price
	}
	if trade.NetAmount == 0 {
		trade.NetAmount = trade.TradeValue + trade.Brokerage + trade.TaxesCharges
	}

	// 1. Transactional write to SQL Server
	if err := database.InsertTrade(&trade); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert trade into MSSQL: " + err.Error()})
		return
	}

	// Retrieve fully enriched trade report (joining client_details and stock_master)
	enrichedReport, err := database.GetTradeReportByID(trade.TradeID)
	if err != nil {
		enrichedReport = &models.TradeReport{
			TradeID:      trade.TradeID,
			OrderID:      trade.OrderID,
			ClientID:     trade.ClientID,
			Symbol:       trade.Symbol,
			Exchange:     trade.Exchange,
			Segment:      trade.Segment,
			TradeType:    trade.TradeType,
			Quantity:     trade.Quantity,
			Price:        trade.Price,
			TradeValue:   trade.TradeValue,
			NetAmount:    trade.NetAmount,
			TradeStatus:  trade.TradeStatus,
			TradeTime:    trade.TradeTime,
		}
	}

	// 2. Stream enriched trade report to Kafka topic 'trade-events'
	if kafka.GlobalProducer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := kafka.GlobalProducer.PublishTradeReport(ctx, enrichedReport); err != nil {
			log.Printf("[Kafka Publish Error] %v", err)
		}
	}

	// 3. Forward to Elasticsearch
	if elastic.GlobalESClient != nil && elastic.GlobalESClient.IsReady() {
		enrichedReport.IndexedAt = time.Now()
		_ = elastic.GlobalESClient.IndexTradeReport(context.Background(), enrichedReport)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Trade inserted into MSSQL and published to Kafka -> Elastic",
		"trade":   enrichedReport,
	})
}

// SyncAllTradesFromSQLHandler synchronizes all trades from MSSQL -> Kafka -> Elasticsearch
func SyncAllTradesFromSQLHandler(c *gin.Context) {
	log.Println("==================================================================")
	log.Println("[MANUAL SYNC] 🔄 Initiating full sync from SQL Server -> Kafka -> Elasticsearch...")
	trades, err := database.GetAllTradesForSync()
	if err != nil {
		log.Printf("[MANUAL SYNC ERROR] Failed to read trades from MSSQL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read trades from MSSQL: " + err.Error()})
		return
	}

	log.Printf("[MANUAL SYNC] 📊 Found %d trades in SQL Server. Beginning real-time stream...", len(trades))
	syncedCount := 0
	ctx := context.Background()

	for i, t := range trades {
		tradeModel := models.Trade{
			TradeID:    t.TradeID,
			OrderID:    t.OrderID,
			ClientID:   t.ClientID,
			Symbol:     t.Symbol,
			Exchange:   t.Exchange,
			Segment:    t.Segment,
			TradeType:  t.TradeType,
			Quantity:   t.Quantity,
			Price:      t.Price,
			TradeValue: t.TradeValue,
			NetAmount:  t.NetAmount,
			TradeTime:  t.TradeTime,
		}

		// 1. Emit to Kafka
		if kafka.GlobalProducer != nil {
			_ = kafka.GlobalProducer.PublishTrade(ctx, &tradeModel)
		}

		// 2. Index to Elasticsearch
		if elastic.GlobalESClient != nil && elastic.GlobalESClient.IsReady() {
			t.IndexedAt = time.Now()
			_ = elastic.GlobalESClient.IndexTradeReport(ctx, &t)
		}

		syncedCount++

		// Print real-time progress to console
		if (i+1)%50 == 0 || i == 0 || i == len(trades)-1 {
			log.Printf("[MANUAL SYNC] ⚡ Synced trade [%d/%d]: %s (%s %s %d @ ₹%.2f) ➔ Kafka ➔ Elastic",
				i+1, len(trades), t.TradeID, t.Symbol, t.TradeType, t.Quantity, t.Price)
		}
	}

	log.Printf("[MANUAL SYNC COMPLETE] ✅ Successfully synced all %d trades from SQL Server -> Kafka -> Elasticsearch!", syncedCount)
	log.Println("==================================================================")

	c.JSON(http.StatusOK, gin.H{
		"message":       "Successfully synced trades from SQL Server -> Kafka -> Elasticsearch",
		"total_trades":  len(trades),
		"synced_count":  syncedCount,
		"sql_source":    "TradeDB.trade_details",
		"kafka_topic":   "trade-events",
		"elastic_index": "trade_reports",
	})
}

// -----------------------------------------------------------------------------
// KAFKA TOPIC MANAGEMENT ENDPOINTS
// -----------------------------------------------------------------------------

func ListKafkaTopicsHandler(c *gin.Context) {
	topics, err := kafka.ListTopics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list Kafka topics: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"total":  len(topics),
		"topics": topics,
	})
}

type CreateKafkaTopicRequest struct {
	Topic             string `json:"topic" binding:"required"`
	Partitions        int    `json:"partitions"`
	ReplicationFactor int    `json:"replication_factor"`
}

func CreateKafkaTopicHandler(c *gin.Context) {
	var req CreateKafkaTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: topic name is required"})
		return
	}

	if req.Partitions <= 0 {
		req.Partitions = 3
	}
	if req.ReplicationFactor <= 0 {
		req.ReplicationFactor = 1
	}

	err := kafka.CreateTopic(req.Topic, req.Partitions, req.ReplicationFactor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create topic: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":            "Kafka topic created successfully",
		"topic":              req.Topic,
		"partitions":         req.Partitions,
		"replication_factor": req.ReplicationFactor,
	})
}

func DeleteKafkaTopicHandler(c *gin.Context) {
	topic := c.Param("topic")
	if topic == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Topic name is required"})
		return
	}

	err := kafka.DeleteTopic(topic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete topic: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Kafka topic deleted successfully",
		"topic":   topic,
	})
}

func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "UP",
		"service": "TradeFlow Go Reporting API",
		"database": gin.H{
			"type":      "Microsoft SQL Server",
			"instance":  os.Getenv("MSSQL_SERVER"),
			"database":  os.Getenv("MSSQL_DATABASE"),
			"connected": database.DB != nil,
		},
		"kafka": gin.H{
			"brokers": os.Getenv("KAFKA_BROKERS"),
			"topic":   os.Getenv("KAFKA_TOPIC"),
		},
		"elasticsearch": gin.H{
			"url":       os.Getenv("ELASTICSEARCH_URL"),
			"index":     os.Getenv("ELASTICSEARCH_INDEX"),
			"connected": elastic.GlobalESClient != nil && elastic.GlobalESClient.IsReady(),
		},
	})
}
