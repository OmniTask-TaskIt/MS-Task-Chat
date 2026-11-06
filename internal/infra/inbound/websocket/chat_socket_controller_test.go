package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/domain/model"
	wsout "github.com/SantiagoSu15/Task-it-Chat-MS/internal/infra/outbound/websocket"
)

type testTokenValidator struct {
	userID string
	err    error
}

func (v testTokenValidator) Validate(string) (string, error) {
	return v.userID, v.err
}

type testHistory struct{}

func (testHistory) GetMessageHistoryByChatID(context.Context, string) ([]model.Message, error) {
	return []model.Message{}, nil
}

type testSender struct {
	senderID string
}

func (s *testSender) SendMessage(_ context.Context, _, _, senderID string) error {
	s.senderID = senderID
	return nil
}

func newTestController(sender *testSender, validator TokenValidator) (*ChatSocketController, *wsout.Client) {
	hub := wsout.NewHub()
	controller := NewChatSocketController(sender, testHistory{}, hub, validator)
	return controller, hub.Register()
}

func TestChatAuthenticateAssociatesJWTIdentity(t *testing.T) {
	sender := &testSender{}
	controller, client := newTestController(sender, testTokenValidator{userID: "sasupu08@gmail.com"})

	ctx, err := controller.handleEvent(context.Background(), client, Event{
		Type:  "chat:authenticate",
		Token: "jwt",
	})
	if err != nil {
		t.Fatal(err)
	}
	userID, ok := authenticatedSender(ctx)
	if !ok || userID != "sasupu08@gmail.com" {
		t.Fatalf("authenticated user = %q, authenticated=%v", userID, ok)
	}
}

func TestChatAuthenticateRejectsInvalidJWT(t *testing.T) {
	sender := &testSender{}
	controller, client := newTestController(sender, testTokenValidator{err: context.Canceled})

	_, err := controller.handleEvent(context.Background(), client, Event{
		Type:  "chat:authenticate",
		Token: "invalid",
	})
	if err != ErrAuthenticationFailed {
		t.Fatalf("error = %v, want %v", err, ErrAuthenticationFailed)
	}
}

func TestEnterRoomRequiresAuthentication(t *testing.T) {
	sender := &testSender{}
	controller, client := newTestController(sender, testTokenValidator{})

	_, err := controller.handleEvent(context.Background(), client, Event{
		Type:   "chat:EnterRoom",
		ChatID: "task-1",
	})
	if err != ErrAuthenticationRequired {
		t.Fatalf("error = %v, want %v", err, ErrAuthenticationRequired)
	}
}

func TestSendMessageRequiresAuthentication(t *testing.T) {
	sender := &testSender{}
	controller, client := newTestController(sender, testTokenValidator{})

	_, err := controller.handleEvent(context.Background(), client, Event{
		Type:    "chat:sendMessage",
		ChatID:  "task-1",
		Content: "hello",
	})
	if err != ErrAuthenticationRequired {
		t.Fatalf("error = %v, want %v", err, ErrAuthenticationRequired)
	}
}

func TestSendMessageUsesJWTIdentity(t *testing.T) {
	sender := &testSender{}
	controller, client := newTestController(sender, testTokenValidator{userID: "sasupu08@gmail.com"})

	ctx, err := controller.handleEvent(context.Background(), client, Event{
		Type:  "chat:authenticate",
		Token: "jwt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controller.handleEvent(ctx, client, Event{
		Type:   "chat:EnterRoom",
		ChatID: "task-1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.handleEvent(ctx, client, Event{
		Type:    "chat:sendMessage",
		ChatID:  "task-1",
		Content: "hello",
	}); err != nil {
		t.Fatal(err)
	}
	if sender.senderID != "sasupu08@gmail.com" {
		t.Fatalf("sender ID = %q, want JWT identity", sender.senderID)
	}
}

func TestChatAuthenticateEnqueuesAcknowledgement(t *testing.T) {
	sender := &testSender{}
	controller, client := newTestController(sender, testTokenValidator{userID: "user-1"})

	_, err := controller.handleEvent(context.Background(), client, Event{
		Type:  "chat:authenticate",
		Token: "jwt",
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case payload := <-client.Messages():
		if string(payload) != `{"type":"chat:authenticated"}` {
			t.Fatalf("payload = %s", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("authentication acknowledgement was not enqueued")
	}
}
