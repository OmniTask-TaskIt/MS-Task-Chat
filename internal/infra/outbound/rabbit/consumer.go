package rabbit

import (
	"context"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/infra/config"
)

type Consumer struct {
	connection *amqp.Connection
	cancel     context.CancelFunc
	done       chan error
}

func NewConsumer(ctx context.Context, cfg config.RabbitConfig, handler *Handler) (*Consumer, error) {
	connection, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}
	if err := declareTopology(channel); err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, err
	}
	if err := channel.Qos(cfg.ConsumerPrefetch, 0, false); err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, fmt.Errorf("configure RabbitMQ QoS: %w", err)
	}

	createMessages, err := channel.Consume(
		config.ChatCreateQueue, cfg.ConsumerName+"-create", false, false, false, false, nil,
	)
	if err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, fmt.Errorf("consume chat create events: %w", err)
	}
	deleteMessages, err := channel.Consume(
		config.ChatDeleteQueue, cfg.ConsumerName+"-delete", false, false, false, false, nil,
	)
	if err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, fmt.Errorf("consume chat delete events: %w", err)
	}

	consumerCtx, cancel := context.WithCancel(ctx)
	consumer := &Consumer{
		connection: connection,
		cancel:     cancel,
		done:       make(chan error, 2),
	}
	go consumer.consume(consumerCtx, createMessages, handler.HandleCreate, "create")
	go consumer.consume(consumerCtx, deleteMessages, handler.HandleDelete, "delete")
	return consumer, nil
}

func (c *Consumer) consume(
	ctx context.Context,
	messages <-chan amqp.Delivery,
	handle func(context.Context, []byte) error,
	eventType string,
) {
	for {
		select {
		case <-ctx.Done():
			c.done <- nil
			return
		case message, ok := <-messages:
			if !ok {
				c.done <- fmt.Errorf("RabbitMQ %s consumer channel closed", eventType)
				return
			}
			if err := handle(ctx, message.Body); err != nil {
				log.Printf("RabbitMQ %s event rejected: %v", eventType, err)
				if rejectErr := message.Reject(false); rejectErr != nil {
					log.Printf("RabbitMQ %s event reject failed: %v", eventType, rejectErr)
				}
				continue
			}
			if err := message.Ack(false); err != nil {
				log.Printf("RabbitMQ %s event acknowledgement failed: %v", eventType, err)
			}
		}
	}
}

func (c *Consumer) Close() error {
	c.cancel()
	err := c.connection.Close()
	<-c.done
	<-c.done
	return err
}

func declareTopology(channel *amqp.Channel) error {
	if err := channel.ExchangeDeclare(config.ChatExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare chat exchange: %w", err)
	}
	if _, err := channel.QueueDeclare(config.ChatCreateQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare chat create queue: %w", err)
	}
	if _, err := channel.QueueDeclare(config.ChatDeleteQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare chat delete queue: %w", err)
	}
	if err := channel.QueueBind(config.ChatCreateQueue, config.RoutingKeyChatCreate, config.ChatExchange, false, nil); err != nil {
		return fmt.Errorf("bind chat create queue: %w", err)
	}
	if err := channel.QueueBind(config.ChatDeleteQueue, config.RoutingKeyChatDelete, config.ChatExchange, false, nil); err != nil {
		return fmt.Errorf("bind chat delete queue: %w", err)
	}
	return nil
}
