package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"orders/internal/models"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Producer implements a Kafka event producer
type Producer struct {
	writer *kafka.Writer
	logger *zap.Logger
	topic  string
}

// NewProducer creates a new Kafka producer instance
func NewProducer(brokers []string, topic string, logger *zap.Logger) *Producer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},    // Use hash to partition by key
		AllowAutoTopicCreation: true,             // Automatically create topic if not exists
		RequiredAcks:           kafka.RequireOne, // At-least-once delivery
		Compression:            kafka.Snappy,     // Compress messages
		MaxAttempts:            3,                // Retry on failure
	}

	return &Producer{
		writer: writer,
		logger: logger,
		topic:  topic,
	}
}

// PublishOrderEvent publishes an order event to Kafka
func (p *Producer) PublishOrderEvent(ctx context.Context, event *models.OrderEvent) error {
	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("Failed to marshal event",
			zap.Error(err),
			zap.String("eventId", event.EventID),
		)
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create Kafka message, using orderID as key to preserve event order per order
	message := kafka.Message{
		Key:   []byte(event.OrderID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "event-type", Value: []byte(event.EventType)},
			{Key: "event-id", Value: []byte(event.EventID)},
		},
	}

	// Publish message
	if err := p.writer.WriteMessages(ctx, message); err != nil {
		p.logger.Error("Failed to publish event",
			zap.Error(err),
			zap.String("eventId", event.EventID),
			zap.String("orderId", event.OrderID),
			zap.String("topic", p.topic),
		)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.logger.Info("Event published successfully",
		zap.String("eventId", event.EventID),
		zap.String("eventType", string(event.EventType)),
		zap.String("orderId", event.OrderID),
		zap.String("topic", p.topic),
	)

	return nil
}

// Close shuts down the Kafka producer
func (p *Producer) Close() error {
	return p.writer.Close()
}
