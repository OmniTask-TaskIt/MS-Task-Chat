package service

import (
	"context"
	"errors"
	"testing"

	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/domain/model"
)

type testChatRepository struct{}

func (testChatRepository) Save(context.Context, model.Chat) (model.Chat, error) {
	return model.Chat{}, nil
}
func (testChatRepository) FindByID(context.Context, string) (model.Chat, error) {
	return model.Chat{ChatID: "chat-1", Active: true}, nil
}
func (testChatRepository) FindByParticipantID(context.Context, string) ([]model.Chat, error) {
	return []model.Chat{}, nil
}
func (testChatRepository) FindByTaskID(context.Context, string) (model.Chat, error) {
	return model.Chat{}, nil
}
func (testChatRepository) Update(context.Context, model.Chat) (model.Chat, error) {
	return model.Chat{}, nil
}
func (testChatRepository) DeleteByID(context.Context, string) error     { return nil }
func (testChatRepository) DeleteByTaskID(context.Context, string) error { return nil }
func (testChatRepository) Exists(context.Context, string) (bool, error) { return true, nil }

type testMessageRepository struct {
	message model.Message
}

func (r *testMessageRepository) Save(_ context.Context, _ string, message model.Message) (model.Message, bool, error) {
	if r.message.MessageID != "" {
		return r.message, false, nil
	}
	r.message = message
	return message, true, nil
}
func (r *testMessageRepository) FindByChatID(context.Context, string) ([]model.Message, error) {
	return []model.Message{r.message}, nil
}
func (r *testMessageRepository) DeleteByChatID(context.Context, string) error { return nil }

type failingMessageRepository struct{}

func (failingMessageRepository) Save(context.Context, string, model.Message) (model.Message, bool, error) {
	return model.Message{}, false, errors.New("redis unavailable")
}
func (failingMessageRepository) FindByChatID(context.Context, string) ([]model.Message, error) {
	return nil, errors.New("redis unavailable")
}
func (failingMessageRepository) DeleteByChatID(context.Context, string) error {
	return errors.New("redis unavailable")
}

type countingChatWebsocket struct {
	calls int
}

func (w *countingChatWebsocket) SendMessage(context.Context, model.Message, string) error {
	w.calls++
	return nil
}

type testChatWebsocket struct {
	message model.Message
	chatID  string
}

func (w *testChatWebsocket) SendMessage(_ context.Context, message model.Message, chatID string) error {
	w.message = message
	w.chatID = chatID
	return nil
}

func TestSendMessageBuildsAndDistributesMessage(t *testing.T) {
	ws := &testChatWebsocket{}
	service := NewSendMessageService(testChatRepository{}, &testMessageRepository{}, ws)

	if err := service.SendMessage(context.Background(), "chat-1", "Hola", "42"); err != nil {
		t.Fatal(err)
	}
	if ws.chatID != "chat-1" || ws.message.ChatID != "chat-1" || ws.message.SenderID != "42" ||
		ws.message.MessageID == "" || ws.message.Content != "Hola" || ws.message.Timestamp.IsZero() {
		t.Fatalf("unexpected message: %+v", ws.message)
	}
}

func TestSendMessageRejectsInvalidContent(t *testing.T) {
	service := NewSendMessageService(testChatRepository{}, &testMessageRepository{}, &testChatWebsocket{})
	tests := []struct {
		name    string
		chatID  string
		content string
		wantErr error
	}{
		{"missing chat", "", "hello", ErrInvalidChatID},
		{"empty content", "chat", "", ErrEmptyContent},
		{"large content", "chat", string(make([]byte, MaxMessageContentLength+1)), ErrMessageTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := service.SendMessage(context.Background(), tt.chatID, tt.content, "1"); err != tt.wantErr {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestSendMessageDoesNotDistributeWhenPersistenceFails(t *testing.T) {
	ws := &countingChatWebsocket{}
	service := NewSendMessageService(testChatRepository{}, failingMessageRepository{}, ws)

	if err := service.SendMessage(context.Background(), "chat-1", "Hola", "42"); err == nil {
		t.Fatal("expected persistence error")
	}
	if ws.calls != 0 {
		t.Fatalf("message was distributed %d times after persistence failure", ws.calls)
	}
}

func TestGetChatsByParticipantIncludesMessages(t *testing.T) {
	messageRepository := &testMessageRepository{
		message: model.Message{
			MessageID: "message-1",
			ChatID:    "chat-1",
			Content:   "Hola",
		},
	}
	service := NewGetMessageHistoryService(
		chatRepositoryWithChat{chat: model.Chat{ChatID: "chat-1"}},
		messageRepository,
	)

	chats, err := service.GetChatsByParticipant(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 1 || len(chats[0].Messages) != 1 || chats[0].Messages[0].MessageID != "message-1" {
		t.Fatalf("unexpected chats: %+v", chats)
	}
}

type chatRepositoryWithChat struct {
	chat model.Chat
}

func (r chatRepositoryWithChat) Save(context.Context, model.Chat) (model.Chat, error) {
	return r.chat, nil
}

func (r chatRepositoryWithChat) FindByID(context.Context, string) (model.Chat, error) {
	return r.chat, nil
}

func (r chatRepositoryWithChat) FindByParticipantID(context.Context, string) ([]model.Chat, error) {
	return []model.Chat{r.chat}, nil
}

func (r chatRepositoryWithChat) FindByTaskID(context.Context, string) (model.Chat, error) {
	return r.chat, nil
}

func (r chatRepositoryWithChat) Update(context.Context, model.Chat) (model.Chat, error) {
	return r.chat, nil
}

func (r chatRepositoryWithChat) DeleteByID(context.Context, string) error     { return nil }
func (r chatRepositoryWithChat) DeleteByTaskID(context.Context, string) error { return nil }
func (r chatRepositoryWithChat) Exists(context.Context, string) (bool, error) { return true, nil }
