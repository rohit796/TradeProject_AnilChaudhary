package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"trade-reporting-backend/models"
)

// Producer publishes keyed domain events to the three TradeFlow topics.
type Producer struct {
	trades  *kafkago.Writer
	clients *kafkago.Writer
	stocks  *kafkago.Writer
}

// GlobalProducer is used by the HTTP handlers.
var GlobalProducer *Producer

func newWriter(topic string) *kafkago.Writer {
	return &kafkago.Writer{
		Addr:     kafkago.TCP(Brokers()...),
		Topic:    topic,
		Balancer: &kafkago.Hash{}, // same key -> same partition -> per-entity ordering
		// Small batch timeout so each synchronous publish returns quickly (default is 1s).
		BatchTimeout:           10 * time.Millisecond,
		WriteTimeout:           5 * time.Second,
		RequiredAcks:           kafkago.RequireOne,
		AllowAutoTopicCreation: true,
	}
}

// InitProducer ensures topics exist and returns a ready producer.
// It always returns a non-nil producer; if Kafka is down, publishes fail with an error instead.
func InitProducer() *Producer {
	if err := EnsureTopics(); err != nil {
		log.Printf("[Kafka] Warning: could not verify topics (%v). Is Kafka running on %v?", err, Brokers())
	}

	p := &Producer{
		trades:  newWriter(TradeTopic()),
		clients: newWriter(TopicClients),
		stocks:  newWriter(TopicStocks),
	}
	GlobalProducer = p
	log.Printf("[Kafka] Producer ready (brokers: %v)", Brokers())
	return p
}

func publish(ctx context.Context, w *kafkago.Writer, key string, v interface{}) error {
	value, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return w.WriteMessages(ctx, kafkago.Message{Key: []byte(key), Value: value, Time: time.Now()})
}

// PublishTrade publishes a raw trade to trade-events (key: trade_id).
func (p *Producer) PublishTrade(ctx context.Context, t *models.Trade) error {
	if p == nil {
		return nil
	}
	return publish(ctx, p.trades, t.TradeID, t)
}

// PublishTradeReport publishes an enriched trade report to trade-events (key: trade_id).
func (p *Producer) PublishTradeReport(ctx context.Context, r *models.TradeReport) error {
	if p == nil {
		return nil
	}
	return publish(ctx, p.trades, r.TradeID, r)
}

// PublishClient publishes a client change to client-events (key: client_id).
func (p *Producer) PublishClient(ctx context.Context, c *models.Client) error {
	if p == nil {
		return nil
	}
	return publish(ctx, p.clients, c.ClientID, c)
}

// PublishStock publishes a stock change to stock-events (key: symbol).
func (p *Producer) PublishStock(ctx context.Context, s *models.Stock) error {
	if p == nil {
		return nil
	}
	return publish(ctx, p.stocks, s.Symbol, s)
}

// Close flushes and closes all writers.
func (p *Producer) Close() {
	if p == nil {
		return
	}
	for _, w := range []*kafkago.Writer{p.trades, p.clients, p.stocks} {
		_ = w.Close()
	}
}
