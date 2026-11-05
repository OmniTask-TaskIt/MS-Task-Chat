package config

import (
	"fmt"
	"os"
)

const (
	TaskExchange = "task.exchange"
	ChatExchange = "chat.exchange"

	ChatCreateQueue = "chat.create.queue"
	ChatDeleteQueue = "chat.delete.queue"

	RoutingKeyChatCreate = "chat.create"
	RoutingKeyChatDelete = "chat.delete"
)

type RabbitConfig struct {
	URL              string
	ConsumerPrefetch int
	ConsumerName     string
}

func RabbitConfigFromEnv() RabbitConfig {
	return RabbitConfig{
		URL:              envOrDefault("RABBITMQ_URL", "amqp://localhost:5672/"),
		ConsumerPrefetch: intOrDefault("RABBITMQ_PREFETCH", 10),
		ConsumerName:     envOrDefault("RABBITMQ_CONSUMER", "chat-service"),
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func intOrDefault(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil || result <= 0 {
		return fallback
	}
	return result
}
