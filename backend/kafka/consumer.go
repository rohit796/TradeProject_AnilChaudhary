package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"trade-reporting-backend/database"
	"trade-reporting-backend/elastic"
	"trade-reporting-backend/models"
)

// referenceCache keeps client and stock master data in memory so trade events
// can be enriched without a SQL round trip for every message.
type referenceCache struct {
	mu      sync.RWMutex
	clients map[string]models.Client
	stocks  map[string]models.Stock
}

var cache = &referenceCache{
	clients: map[string]models.Client{},
	stocks:  map[string]models.Stock{},
}

func (c *referenceCache) load() {
	clients, _ := database.GetClients()
	stocks, _ := database.GetStocks()

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, cl := range clients {
		c.clients[cl.ClientID] = cl
	}
	for _, s := range stocks {
		c.stocks[s.Symbol] = s
	}
	log.Printf("[Kafka Consumer] Loaded %d clients and %d stocks into local reference cache", len(c.clients), len(c.stocks))
}

// enrich fills client/stock fields that are missing on a trade event.
func (c *referenceCache) enrich(r *models.TradeReport) {
	if r.ClientCode != "" && r.CompanyName != "" {
		return
	}
	c.mu.RLock()
	cl, okC := c.clients[r.ClientID]
	st, okS := c.stocks[r.Symbol]
	c.mu.RUnlock()

	if !okC || !okS {
		// Unknown entity (cache is stale): read the enriched row from SQL instead.
		if full, err := database.GetTradeReportByID(r.TradeID); err == nil {
			*r = *full
			return
		}
	}
	if okC {
		r.ClientCode, r.FirstName, r.LastName, r.AccountType = cl.ClientCode, cl.FirstName, cl.LastName, cl.AccountType
	}
	if okS {
		r.CompanyName, r.Sector = st.CompanyName, st.Sector
	}
}

func groupID(suffix string) string {
	base := os.Getenv("KAFKA_GROUP_ID")
	if base == "" {
		base = "trade-reporting-group"
	}
	return base + "-" + suffix
}

// consume reads a topic in a consumer group until ctx is cancelled.
func consume(ctx context.Context, topic, group string, handle func(kafkago.Message)) {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:     Brokers(),
		Topic:       topic,
		GroupID:     group,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     500 * time.Millisecond,
		StartOffset: kafkago.FirstOffset,
	})
	defer reader.Close()

	for {
		msg, err := reader.ReadMessage(ctx) // commits the offset automatically (consumer group)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[Kafka Consumer] %s read error: %v (retrying)", topic, err)
			time.Sleep(2 * time.Second)
			continue
		}
		handle(msg)
	}
}

// StartConsumer starts one goroutine per topic:
//   - trade-events  -> enrich from the reference cache -> index into Elasticsearch
//   - client-events -> refresh client cache
//   - stock-events  -> refresh stock cache
func StartConsumer(ctx context.Context, es *elastic.ESClient) error {
	cache.load()

	go consume(ctx, TradeTopic(), groupID("trades"), func(m kafkago.Message) {
		var r models.TradeReport
		if err := json.Unmarshal(m.Value, &r); err != nil || r.TradeID == "" {
			log.Printf("[Kafka Consumer] Skipping malformed trade event (key=%s)", string(m.Key))
			return
		}
		cache.enrich(&r)
		if es == nil || !es.IsReady() {
			return
		}
		r.IndexedAt = time.Now()
		if err := es.IndexTradeReport(ctx, &r); err != nil {
			log.Printf("[Kafka Consumer] Failed to index %s: %v", r.TradeID, err)
		}
	})

	go consume(ctx, TopicClients, groupID("clients"), func(m kafkago.Message) {
		var c models.Client
		if err := json.Unmarshal(m.Value, &c); err != nil || c.ClientID == "" {
			return
		}
		cache.mu.Lock()
		cache.clients[c.ClientID] = c
		cache.mu.Unlock()
	})

	go consume(ctx, TopicStocks, groupID("stocks"), func(m kafkago.Message) {
		var s models.Stock
		if err := json.Unmarshal(m.Value, &s); err != nil || s.Symbol == "" {
			return
		}
		cache.mu.Lock()
		cache.stocks[s.Symbol] = s
		cache.mu.Unlock()
	})

	log.Printf("[Kafka Consumer] Goroutines started on %s, %s, %s", TradeTopic(), TopicClients, TopicStocks)
	return nil
}
