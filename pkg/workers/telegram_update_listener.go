package workers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type Handler interface {
	HandleUpdate(ctx context.Context, update *tgbotapi.Update)
}

type Authenticator interface {
	IsAuthorized(userID int64) bool
}

type TelegramClient interface {
	GetUpdates() tgbotapi.UpdatesChannel
	SendResponse(ctx context.Context, response *domain.Response)
	AcknowledgeCallback(ctx context.Context, callbackQueryID string)
	StartTyping(ctx context.Context, chatID int64)
}

type telegramUpdateListener struct {
	client        TelegramClient
	authenticator Authenticator
	handler       Handler
	responseCh    <-chan domain.Response
	wg            sync.WaitGroup
}

func NewTelegramUpdateListener(
	client TelegramClient,
	authenticator Authenticator,
	handler Handler,
	responseCh <-chan domain.Response,
) (*telegramUpdateListener, error) {
	return &telegramUpdateListener{
		client:        client,
		authenticator: authenticator,
		handler:       handler,
		responseCh:    responseCh,
	}, nil
}

func (t *telegramUpdateListener) Name() string { return "telegram_listener_worker" }

func (t *telegramUpdateListener) Start(ctx context.Context) error {
	slog.Info("Starting worker", "name", t.Name())
	defer slog.Info("Worker stopped", "name", t.Name())

	updates := t.client.GetUpdates()

	for {
		select {
		case <-ctx.Done():
			t.wg.Wait()
			return nil
		case update := <-updates:
			t.wg.Add(1)
			go func(update tgbotapi.Update) {
				defer t.wg.Done()
				t.processUpdate(ctx, &update)
			}(update)
		case response := <-t.responseCh:
			t.client.SendResponse(ctx, &response)
		}
	}
}

func (t *telegramUpdateListener) processUpdate(ctx context.Context, update *tgbotapi.Update) {
	ctx = logger.ContextWithRequestID(ctx, update.UpdateID)

	var chatID, userID int64
	switch {
	case update.Message != nil:
		chatID, userID = update.Message.Chat.ID, update.Message.From.ID
	case update.CallbackQuery != nil:
		chatID, userID = update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.From.ID
		defer t.client.AcknowledgeCallback(ctx, update.CallbackQuery.ID)
	default:
		slog.WarnContext(ctx, "Received unknown update type", "update", update)
		return
	}

	slog.InfoContext(ctx, "Processing update", "chatID", chatID, "userID", userID)

	t.client.StartTyping(ctx, chatID)

	if !t.authenticator.IsAuthorized(userID) {
		slog.WarnContext(ctx, "Unauthorized access attempt")
		t.client.SendResponse(ctx, &domain.Response{
			ChatID: chatID,
			Text:   fmt.Sprintf("User PromptID %d is not authorized", userID),
		})
		return
	}

	t.handler.HandleUpdate(ctx, update)
}
