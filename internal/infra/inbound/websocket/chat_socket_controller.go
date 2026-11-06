package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"

	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/application/port/inbound"
	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/application/service"
	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/domain/model"
	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/infra/config"
	wsout "github.com/SantiagoSu15/Task-it-Chat-MS/internal/infra/outbound/websocket"
)

const (
	maxWebSocketMessageSize = 64 * 1024
	writeTimeout            = 5 * time.Second
)

var (
	ErrAuthenticationRequired = errors.New("authentication is required")
	ErrAuthenticationFailed   = errors.New("invalid authentication token")
)

type TokenValidator interface {
	Validate(token string) (string, error)
}

type ChatSocketController struct {
	sendMessage inbound.SendMessageUseCase
	history     inbound.GetChatHistoryUseCase
	hub         *wsout.Hub
	tokens      TokenValidator
}

func NewChatSocketController(
	sendMessage inbound.SendMessageUseCase,
	history inbound.GetChatHistoryUseCase,
	hub *wsout.Hub,
	tokens ...TokenValidator,
) *ChatSocketController {
	var tokenValidator TokenValidator
	if len(tokens) > 0 {
		tokenValidator = tokens[0]
	}
	return &ChatSocketController{
		sendMessage: sendMessage,
		history:     history,
		hub:         hub,
		tokens:      tokenValidator,
	}
}

func (c *ChatSocketController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.hub == nil || c.sendMessage == nil || c.history == nil {
		http.Error(w, "WebSocket controller is not configured", http.StatusInternalServerError)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "chat WebSocket closed")

	client := c.hub.Register()
	defer c.hub.Unregister(client)

	ctx := withAuthenticatedSender(r.Context(), r.Header.Get("X-User-ID"))
	go c.writeLoop(ctx, conn, client)
	conn.SetReadLimit(maxWebSocketMessageSize)

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var event Event
		if err := json.Unmarshal(data, &event); err != nil {
			c.writeError(client, "invalid JSON event")
			continue
		}
		nextCtx, err := c.handleEvent(ctx, client, event)
		ctx = nextCtx
		if err != nil {
			if errors.Is(err, ErrAuthenticationFailed) || errors.Is(err, ErrAuthenticationRequired) {
				message := "invalid authentication token"
				if errors.Is(err, ErrAuthenticationRequired) {
					message = "authentication is required"
				}
				c.writeError(client, message)
				_ = conn.Close(websocket.StatusPolicyViolation, "authentication failed")
				return
			}
			c.writeError(client, protocolError(err))
		}
	}
}

type Event struct {
	Type      string `json:"type"`
	Token     string `json:"token"`
	ChatID    string `json:"chatId"`
	Content   string `json:"content"`
	MessageID string `json:"messageId"`
}

func (c *ChatSocketController) handleEvent(
	ctx context.Context,
	client *wsout.Client,
	event Event,
) (context.Context, error) {
	switch event.Type {
	case "chat:authenticate":
		if c.tokens == nil {
			return ctx, ErrAuthenticationFailed
		}
		userID, err := c.tokens.Validate(event.Token)
		if err != nil || strings.TrimSpace(userID) == "" {
			return ctx, ErrAuthenticationFailed
		}
		authenticatedCtx := withAuthenticatedSender(ctx, userID)
		if !client.Enqueue([]byte(`{"type":"chat:authenticated"}`)) {
			c.hub.Unregister(client)
		}
		return authenticatedCtx, nil
	case "chat:EnterRoom":
		if _, ok := authenticatedSender(ctx); !ok {
			return ctx, ErrAuthenticationRequired
		}
		chatID := strings.TrimSpace(event.ChatID)
		if chatID == "" {
			return ctx, wsout.ErrInvalidRoom
		}
		history, err := c.history.GetMessageHistoryByChatID(ctx, chatID)
		if err != nil {
			return ctx, err
		}
		if err := c.hub.Join(client, chatID); err != nil {
			return ctx, err
		}
		payload, err := json.Marshal(struct {
			Type string          `json:"type"`
			Data []model.Message `json:"data"`
		}{Type: "chat:history", Data: history})
		if err != nil {
			return ctx, err
		}
		if !client.Enqueue(payload) {
			c.hub.Unregister(client)
		}
		return ctx, nil
	case "chat:sendMessage":
		if _, ok := authenticatedSender(ctx); !ok {
			return ctx, ErrAuthenticationRequired
		}
		if !c.hub.InRoom(client, strings.TrimSpace(event.ChatID)) {
			return ctx, errors.New("client must enter the room before sending")
		}
		senderID, _ := authenticatedSender(ctx)
		return ctx, c.sendMessage.SendMessage(ctx, event.ChatID, event.MessageID, event.Content, senderID)
	default:
		return ctx, errors.New("unsupported event type")
	}
}

func (c *ChatSocketController) writeLoop(ctx context.Context, conn *websocket.Conn, client *wsout.Client) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-client.Done():
			return
		case payload := <-client.Messages():
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := conn.Write(writeCtx, websocket.MessageText, payload)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (c *ChatSocketController) writeError(client *wsout.Client, message string) {
	payload, _ := json.Marshal(map[string]string{"type": "error", "error": message})
	if !client.Enqueue(payload) {
		c.hub.Unregister(client)
	}
}

type senderContextKey struct{}

func withAuthenticatedSender(ctx context.Context, value string) context.Context {
	id := strings.TrimSpace(value)
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, senderContextKey{}, id)
}

func authenticatedSender(ctx context.Context) (string, bool) {
	senderID, ok := ctx.Value(senderContextKey{}).(string)
	return strings.TrimSpace(senderID), ok && strings.TrimSpace(senderID) != ""
}

func protocolError(err error) string {
	switch {
	case errors.Is(err, service.ErrInvalidChatID), errors.Is(err, service.ErrInvalidMessage),
		errors.Is(err, service.ErrEmptyContent), errors.Is(err, service.ErrMessageTooLarge):
		return err.Error()
	case errors.Is(err, ErrAuthenticationRequired):
		return "authentication is required before entering a room or sending messages"
	case errors.Is(err, config.ErrInvalidJWT), errors.Is(err, config.ErrJWTSecretMissing),
		errors.Is(err, ErrAuthenticationFailed):
		return "invalid authentication token"
	default:
		return err.Error()
	}
}
