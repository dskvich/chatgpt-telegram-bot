package workers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type Registry interface {
	HandleUpdate(ctx context.Context, update *tgbotapi.Update)
}

type Authenticator interface {
	IsAuthorized(userID int64) bool
}

type TelegramClient interface {
	GetUpdates() tgbotapi.UpdatesChannel
	SendResponse(ctx context.Context, chatID int64, response *domain.Response)
}

type telegramUpdateListener struct {
	client          TelegramClient
	authenticator   Authenticator
	registry        Registry
	poolSize        int
	pollingInterval time.Duration
}

func NewTelegramUpdateListener(
	client TelegramClient,
	authenticator Authenticator,
	registry Registry,
	poolSize int,
	pollingInterval time.Duration,
) (*telegramUpdateListener, error) {
	return &telegramUpdateListener{
		client:          client,
		authenticator:   authenticator,
		registry:        registry,
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
			go func(update *tgbotapi.Update) {
				defer func() { <-pool }()
				t.processUpdate(update)
			}(&update)
		default:
			time.Sleep(t.pollingInterval)
		}
	}
}

func (t *telegramUpdateListener) processUpdate(update *tgbotapi.Update) {
	ctx := logger.ContextWithRequestID(context.Background(), update.UpdateID)

	var chatID, userID int64
	switch {
	case update.Message != nil:
		chatID, userID = update.Message.Chat.ID, update.Message.From.ID
	case update.CallbackQuery != nil:
		chatID, userID = update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.From.ID
	default:
		slog.WarnContext(ctx, "Unknown update type", "update", update)
		return
	}

	slog.InfoContext(ctx, "Processing update", "chatID", chatID, "userID", userID)

	if !t.authenticator.IsAuthorized(userID) {
		slog.WarnContext(ctx, "Unauthorized access attempt")

		t.respondUnauthorized(ctx, chatID, userID)
		return
	}

	t.registry.HandleUpdate(ctx, update)
}

func (t *telegramUpdateListener) respondUnauthorized(ctx context.Context, chatID, userID int64) {
	text := fmt.Sprintf("User ID %d is not authorized", userID)
	t.client.SendResponse(ctx, chatID, &domain.Response{Text: text})
}
