package rabbit

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/infra/config"
	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/domain/model"
)

type MessagePayload struct {
	MessageID string    `json:"message_id"`
	ChatID    string    `json:"chat_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type ChatMessageSentEvent struct {
	EventID    string         `json:"event_id"`
	EventType  string         `json:"event_type"`
	OccurredAt time.Time      `json:"occurred_at"`
	TaskID     string         `json:"task_id"`
	Message    MessagePayload `json:"message"`
}

type EventPublisher struct {
	channel *amqp.Channel
}

func NewEventPublisher(ch *amqp.Channel) *EventPublisher {
	return &EventPublisher{channel: ch}
}

func (p *EventPublisher) PublishChatMessage(taskID string, msg model.Message) error {
	event := ChatMessageSentEvent{
		EventID:    "evt-" + uuid.New().String(),
		EventType:  "ChatMessageSentEvent",
		OccurredAt: time.Now().UTC(),
		TaskID:     taskID,
		Message: MessagePayload{
			MessageID: msg.MessageID,
			ChatID:    msg.ChatID,
			SenderID:  msg.SenderID,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
		},
	}

	bodyJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("[ERROR] [RABBITMQ] Error serializando ChatMessageSentEvent: %v", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(ctx,
		config.EventsExchange,       
		config.RoutingKeyAitaEvents,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         bodyJSON,
		},
	)

	if err != nil {
		log.Printf("[ERROR] [RABBITMQ] Fallo enviando evento a AITA (msg_id=%s): %v", msg.MessageID, err)
		return err
	}

	log.Printf("[INFO] [RABBITMQ] Evento ChatMessageSentEvent publicado hacia AITA con éxito (event_id: %s)", event.EventID)
	return nil
}
