package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"orders/internal/models"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Producer implementa el productor de eventos de Kafka
type Producer struct {
	writer *kafka.Writer
	logger *zap.Logger
	topic  string
}

// NewProducer crea una nueva instancia del productor
func NewProducer(brokers []string, topic string, logger *zap.Logger) *Producer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{}, // Usar hash para particionar por key
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireOne, // At-least-once delivery
		Compression:            kafka.Snappy,
		MaxAttempts:            3,
	}

	return &Producer{
		writer: writer,
		logger: logger,
		topic:  topic,
	}
}

// PublishOrderEvent publica un evento de orden
func (p *Producer) PublishOrderEvent(ctx context.Context, event *models.OrderEvent) error {
	// Serializar evento a JSON
	data, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("Failed to marshal event",
			zap.Error(err),
			zap.String("eventId", event.EventID),
		)
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Crear mensaje de Kafka
	// Usar orderID como key para mantener orden de eventos por orden
	message := kafka.Message{
		Key:   []byte(event.OrderID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "event-type", Value: []byte(event.EventType)},
			{Key: "event-id", Value: []byte(event.EventID)},
		},
	}

	// Publicar mensaje
	err = p.writer.WriteMessages(ctx, message)
	if err != nil {
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

// Close cierra el productor
func (p *Producer) Close() error {
	return p.writer.Close()
}
