package inbound

import "context"

type SendMessageUseCase interface {
	SendMessage(ctx context.Context, chatID, content, senderID string) error
}
