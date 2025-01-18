package workers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type Handler interface {
	CanHandle(*tgbotapi.Update) bool
	Handle(*tgbotapi.Update)
}

type Authenticator interface {
	IsAuthorized(userID int64) bool
}

type TelegramClient interface {
	GetUpdates() tgbotapi.UpdatesChannel
	SendTextMessage(domain.TextMessage)
}

type telegramUpdateListener struct {
	client          TelegramClient
	authenticator   Authenticator
	handlers        []Handler
	poolSize        int
	pollingInterval time.Duration
}

func NewTelegramUpdateListener(
	client TelegramClient,
	authenticator Authenticator,
	handlers []Handler,
	poolSize int,
	pollingInterval time.Duration,
) (*telegramUpdateListener, error) {
	return &telegramUpdateListener{
		client:          client,
		authenticator:   authenticator,
		handlers:        handlers,
		poolSize:        poolSize,
		pollingInterval: pollingInterval,
	}, nil
}

func (t *telegramUpdateListener) Name() string { return "telegram_listener_worker" }

func (t *telegramUpdateListener) Start(ctx context.Context) error {
	slog.Info("Starting worker", "name", t.Name())
	defer slog.Info("Worker stopped", "name", t.Name())

	pool := make(chan struct{}, t.poolSize)

	for {
		select {
		case <-ctx.Done():
			close(pool)
			return nil
		case update := <-t.client.GetUpdates():
			pool <- struct{}{}
			go func(update tgbotapi.Update) {
				defer func() { <-pool }()
				t.processUpdate(update)
			}(update)
		default:
			time.Sleep(t.pollingInterval)
		}
	}
}

func (t *telegramUpdateListener) processUpdate(update tgbotapi.Update) {
	var chatID, userID int64
	if update.Message != nil {
		chatID, userID = update.Message.Chat.ID, update.Message.From.ID
	} else if update.CallbackQuery != nil {
		chatID, userID = update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.From.ID
	} else {
		slog.Warn("Unknown update type", "update", update)
		return
	}

	slog.Info("Processing update", "update", update)

	if !t.authenticator.IsAuthorized(userID) {
		t.respondUnauthorized(chatID, userID)
		return
	}

	for _, h := range t.handlers {
		if h.CanHandle(&update) {
			h.Handle(&update)
			return
		}
	}

	slog.Warn("No handler found for update", "update", update)
}

func (t *telegramUpdateListener) respondUnauthorized(chatID, userID int64) {
	slog.Warn("Unauthorized access attempt", "userID", userID, "chatID", chatID)

	t.client.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   fmt.Sprintf("User ID %d is not authorized", userID),
	})
}
