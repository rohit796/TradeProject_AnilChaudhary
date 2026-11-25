package kafka

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// Domain topics, one per SQL table.
const (
	TopicTrades  = "trade-events"  // trade_details  (key: trade_id)
	TopicClients = "client-events" // client_details (key: client_id)
	TopicStocks  = "stock-events"  // stock_master   (key: symbol)
)

// TopicInfo is returned by the Kafka topic management API.
type TopicInfo struct {
	Name       string `json:"name"`
	Partitions int    `json:"partitions"`
}

var dialer = &kafkago.Dialer{Timeout: 5 * time.Second}

// Brokers returns the broker list from KAFKA_BROKERS (comma separated), defaulting to localhost:9092.
func Brokers() []string {
	raw := os.Getenv("KAFKA_BROKERS")
	if raw == "" {
		raw = "localhost:9092"
	}
	var out []string
	for _, b := range strings.Split(raw, ",") {
		if b = strings.TrimSpace(b); b != "" {
			out = append(out, b)
		}
	}
	return out
}

// TradeTopic returns the trade topic name (KAFKA_TOPIC overrides the default).
func TradeTopic() string {
	if t := os.Getenv("KAFKA_TOPIC"); t != "" {
		return t
	}
	return TopicTrades
}

func dialAny() (*kafkago.Conn, error) {
	var lastErr error
	for _, b := range Brokers() {
		conn, err := dialer.Dial("tcp", b)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("no Kafka broker reachable at %v: %w", Brokers(), lastErr)
}

func dialController() (*kafkago.Conn, error) {
	conn, err := dialAny()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return nil, fmt.Errorf("could not find Kafka controller: %w", err)
	}
	return dialer.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
}

// CreateTopic creates a topic; an already existing topic is not an error.
func CreateTopic(name string, partitions, replicationFactor int) error {
	conn, err := dialController()
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.CreateTopics(kafkago.TopicConfig{
		Topic:             name,
		NumPartitions:     partitions,
		ReplicationFactor: replicationFactor,
	})
	if err != nil && !errors.Is(err, kafkago.TopicAlreadyExists) {
		return err
	}
	return nil
}

// DeleteTopic deletes a topic.
func DeleteTopic(name string) error {
	conn, err := dialController()
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.DeleteTopics(name)
}

// ListTopics lists user topics (internal "__" topics are hidden) with their partition counts.
func ListTopics() ([]TopicInfo, error) {
	conn, err := dialAny()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, err
	}

	counts := map[string]int{}
	for _, p := range partitions {
		if strings.HasPrefix(p.Topic, "__") {
			continue
		}
		counts[p.Topic]++
	}

	topics := make([]TopicInfo, 0, len(counts))
	for name, n := range counts {
		topics = append(topics, TopicInfo{Name: name, Partitions: n})
	}
	sort.Slice(topics, func(i, j int) bool { return topics[i].Name < topics[j].Name })
	return topics, nil
}

// EnsureTopics creates the three TradeFlow topics if they are missing.
func EnsureTopics() error {
	for _, t := range []string{TradeTopic(), TopicClients, TopicStocks} {
		if err := CreateTopic(t, 3, 1); err != nil {
			return fmt.Errorf("topic %s: %w", t, err)
		}
		log.Printf("[Kafka Admin] Topic '%s' ready", t)
	}
	return nil
}
