package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/onlyspans/events/internal/config"
	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/ports"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	eventsReceivedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "events_received_total",
		Help: "Total number of events received from Kafka",
	})

	batchesProcessedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "events_batches_processed_total",
		Help: "Total number of batches processed",
	})

	eventsFailedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "events_failed_total",
		Help: "Total number of events that failed to process",
	})
)

const (
	// batchSize is the maximum number of messages to process in a single batch.
	batchSize = 100
	// batchTimeout is the maximum time to wait before processing a partial batch.
	batchTimeout = 500 * time.Millisecond
)

// KafkaConsumer handles consuming events from Kafka.
type KafkaConsumer struct {
	consumer sarama.ConsumerGroup
	ingester ports.EventIngester
	topic    string
	logger   *slog.Logger
}

// NewKafkaConsumer creates a new KafkaConsumer.
func NewKafkaConsumer(cfg *config.KafkaConfig, ingester ports.EventIngester, logger *slog.Logger) (*KafkaConsumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V3_6_0_0
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaConfig.Consumer.MaxProcessingTime = 30 * time.Second
	saramaConfig.Consumer.Return.Errors = true

	saramaConfig.Consumer.Offsets.AutoCommit.Enable = false

	saramaConfig.Consumer.Fetch.Min = 1
	saramaConfig.Consumer.MaxWaitTime = 500 * time.Millisecond
	saramaConfig.Consumer.Group.Session.Timeout = 30 * time.Second
	saramaConfig.Consumer.Group.Heartbeat.Interval = 10 * time.Second

	if cfg.Username != "" && cfg.Password != "" {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		saramaConfig.Net.SASL.User = cfg.Username
		saramaConfig.Net.SASL.Password = cfg.Password
		saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &scramClient{HashGeneratorFcn: SHA512}
		}
	}

	brokers := cfg.GetBrokers()
	consumer, err := sarama.NewConsumerGroup(brokers, cfg.GroupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	return &KafkaConsumer{
		consumer: consumer,
		ingester: ingester,
		topic:    cfg.Topic,
		logger:   logger,
	}, nil
}

// Start begins consuming messages from Kafka.
// It blocks until the context is cancelled or an unrecoverable error occurs.
func (c *KafkaConsumer) Start(ctx context.Context) error {
	// Create ready channel once, outside the loop
	ready := make(chan struct{})

	handler := &consumerGroupHandler{
		consumer: c,
		ready:    ready,
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			// Consume should be called inside an infinite loop
			// When a server-side rebalance happens, the consumer session will need to be recreated
			if err := c.consumer.Consume(ctx, []string{c.topic}, handler); err != nil {
				c.logger.Error("error from consumer", "error", err)
			}

			// Check if context was cancelled
			if ctx.Err() != nil {
				c.logger.Info("context cancelled, stopping consumer")
				return
			}

			// Reset ready channel for next session (safe because previous is closed)
			handler.resetReady()
		}
	}()

	// Wait for consumer to be ready (first session established)
	select {
	case <-ready:
		c.logger.Info("kafka consumer started", "topic", c.topic)
	case <-ctx.Done():
		c.logger.Info("context cancelled before consumer ready")
		return ctx.Err()
	}

	// Handle errors in a separate goroutine
	go func() {
		for err := range c.consumer.Errors() {
			c.logger.Error("consumer error", "error", err)
		}
	}()

	wg.Wait()
	return nil
}

// Close closes the Kafka consumer.
func (c *KafkaConsumer) Close() error {
	if err := c.consumer.Close(); err != nil {
		return fmt.Errorf("failed to close consumer: %w", err)
	}
	c.logger.Info("kafka consumer closed")
	return nil
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler.
type consumerGroupHandler struct {
	consumer *KafkaConsumer
	ready    chan struct{}
	mu       sync.Mutex
}

// resetReady creates a new ready channel for the next session.
// This must be called after a session ends and before the next one starts.
func (h *consumerGroupHandler) resetReady() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ready = make(chan struct{})
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Close ready channel to signal that the consumer is ready
	select {
	case <-h.ready:
		// Already closed
	default:
		close(h.ready)
	}
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// Use session context for proper cancellation propagation
	ctx := session.Context()

	batch := make([]*eventWithMessage, 0, batchSize)
	ticker := time.NewTicker(batchTimeout)
	defer ticker.Stop()

	processBatch := func() error {
		if len(batch) == 0 {
			return nil
		}

		h.consumer.logger.Info("processing batch", "size", len(batch))
		eventsReceivedCounter.Add(float64(len(batch)))

		// Convert to DTOs for service
		dtos := make([]*dto.EventDTO, len(batch))
		for i, ewm := range batch {
			dtos[i] = &ewm.EventDTO
		}

		// Use session context for the ingestion call
		if err := h.consumer.ingester.IngestEvents(ctx, dtos); err != nil {
			h.consumer.logger.Error("failed to ingest batch", "error", err, "size", len(batch))
			return err
		}

		batchesProcessedCounter.Inc()
		h.consumer.logger.Info("successfully processed batch", "size", len(batch))

		// Mark last message in batch as processed (commits all previous messages too)
		lastIdx := len(batch) - 1
		session.MarkMessage(batch[lastIdx].msg, "")
		session.Commit()

		batch = make([]*eventWithMessage, 0, batchSize)
		return nil
	}

	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			h.consumer.logger.Debug("received message",
				"topic", message.Topic,
				"partition", message.Partition,
				"offset", message.Offset)

			var event dto.EventDTO
			if err := json.Unmarshal(message.Value, &event); err != nil {
				h.consumer.logger.Error("failed to parse message",
					"error", err,
					"offset", message.Offset,
					"partition", message.Partition,
					"message", string(message.Value))
				eventsFailedCounter.Inc()
				// Mark message as processed even if parsing fails to avoid reprocessing
				session.MarkMessage(message, "")
				continue
			}

			// Wrap message to track original for commit
			batch = append(batch, &eventWithMessage{EventDTO: event, msg: message})

			// Process batch when it reaches size limit
			if len(batch) >= batchSize {
				if err := processBatch(); err != nil {
					// Don't commit on error - messages will be reprocessed
					return err
				}
			}

		case <-ticker.C:
			// Process batch on timer to avoid waiting too long for partial batches
			if err := processBatch(); err != nil {
				return err
			}

		case <-ctx.Done():
			// Process remaining batch before shutdown
			if err := processBatch(); err != nil {
				h.consumer.logger.Error("failed to process final batch on shutdown", "error", err)
			}
			return nil
		}
	}
}

// eventWithMessage wraps an event DTO with its Kafka message for commit tracking.
type eventWithMessage struct {
	dto.EventDTO
	msg *sarama.ConsumerMessage
}
