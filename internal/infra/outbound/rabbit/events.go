package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/domain/model"
	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/domain/repository"
)

type CreateChatEvent struct {
	TaskID       string `json:"taskId"`
	TaskName     string `json:"taskName"`
	TaskOwner    string `json:"taskOwner"`
	TaskAssignee string `json:"taskAssignee"`
}

type DeleteChatEvent struct {
	TaskID string `json:"taskId"`
}

type Handler struct {
	chats repository.ChatRepository
}

func NewHandler(chats repository.ChatRepository) *Handler {
	return &Handler{chats: chats}
}

func (h *Handler) HandleCreate(ctx context.Context, body []byte) error {
	var event CreateChatEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("decode chat create event: %w", err)
	}
	if strings.TrimSpace(event.TaskID) == "" {
		return errors.New("chat create event taskId is required")
	}
	if strings.TrimSpace(event.TaskOwner) == "" {
		return errors.New("chat create event taskOwner is required")
	}
	if strings.TrimSpace(event.TaskAssignee) == "" {
		return errors.New("chat create event taskAssignee is required")
	}

	var buyerID *int64
	if parsedBuyerID, err := strconv.ParseInt(event.TaskAssignee, 10, 64); err == nil {
		buyerID = &parsedBuyerID
	}

	_, err := h.chats.Save(ctx, model.Chat{
		ChatID:    event.TaskID,
		SellerID:  event.TaskOwner,
		BuyerID:   buyerID,
		Active:    true,
		CreatedAt: time.Now().UTC(),
		TaskID:    event.TaskID,
	})
	if err != nil {
		return fmt.Errorf("save chat %q: %w", event.TaskID, err)
	}
	return nil
}

func (h *Handler) HandleDelete(ctx context.Context, body []byte) error {
	var event DeleteChatEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("decode chat delete event: %w", err)
	}
	if strings.TrimSpace(event.TaskID) == "" {
		return errors.New("chat delete event taskId is required")
	}
	if err := h.chats.DeleteByTaskID(ctx, event.TaskID); err != nil {
		return fmt.Errorf("delete chat for task %q: %w", event.TaskID, err)
	}
	return nil
}
