package inbound

import "context"

type SendMessageUseCase interface {
	SendMessage(ctx context.Context, chatID, messageID, content, senderID string) error
}
