package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/application/port/inbound"
	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/application/port/outbound"
	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/domain/model"
	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/domain/repository"
)

var (
	ErrInvalidChatID   = errors.New("chatId is required")
	ErrEmptyContent    = errors.New("message content is required")
	ErrMessageTooLarge = errors.New("message content exceeds the maximum size")
)

const MaxMessageContentLength = 4096

type SendMessageService struct {
	repository        repository.ChatRepository
	messageRepository repository.MessageRepository
	websocket         outbound.ChatWs
}

func NewSendMessageService(chatRepository repository.ChatRepository, messageRepository repository.MessageRepository, chatWebsocket outbound.ChatWs) *SendMessageService {
	return &SendMessageService{
		repository:        chatRepository,
		messageRepository: messageRepository,
		websocket:         chatWebsocket,
	}
}

func (s *SendMessageService) SendMessage(ctx context.Context, chatID, content, senderID string) error {
	chatID = strings.TrimSpace(chatID)
	content = strings.TrimSpace(content)
	if chatID == "" {
		return ErrInvalidChatID
	}
	if content == "" {
		return ErrEmptyContent
	}
	if len([]rune(content)) > MaxMessageContentLength {
		return ErrMessageTooLarge
	}
	senderID = strings.TrimSpace(senderID)
	if senderID == "" {
		return errors.New("authenticated sender is required")
	}

	if s.repository == nil {
		return errors.New("chat repository is not configured")
	}
	if s.messageRepository == nil {
		return errors.New("message repository is not configured")
	}
	if s.websocket == nil {
		return errors.New("chat websocket is not configured")
	}
	if _, err := s.repository.FindByID(ctx, chatID); err != nil {
		return err
	}

	messageID, err := newMessageID()
	if err != nil {
		return err
	}
	message := model.Message{
		MessageID: messageID,
		ChatID:    chatID,
		SenderID:  senderID,
		Content:   content,
		Timestamp: time.Now().UTC(),
	}
	persisted, created, err := s.messageRepository.Save(ctx, chatID, message)
	if err != nil {
		log.Printf("[chat][message] persist failed chatId=%s messageId=%s error=%v", chatID, messageID, err)
		return err
	}
	if !created {
		return nil
	}
	log.Printf("[chat][message] persisted chatId=%s messageId=%s", chatID, persisted.MessageID)
	return s.websocket.SendMessage(ctx, persisted, chatID)
}

func newMessageID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", fmt.Errorf("generate message id: %w", err)
	}
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		id[0:4], id[4:6], id[6:8], id[8:10], id[10:16]), nil
}

var _ inbound.SendMessageUseCase = (*SendMessageService)(nil)
