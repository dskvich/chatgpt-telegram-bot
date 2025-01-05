package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type Authenticator interface {
	IsAuthorized(userID int64) bool
}

type TelegramClient interface {
	GetUpdates() tgbotapi.UpdatesChannel
	SendTextMessage(domain.TextMessage)
}

type CommandHandler interface {
	Handle(update tgbotapi.Update)
}

type telegramListener struct {
	client         TelegramClient
	authenticator  Authenticator
	commandHandler CommandHandler
}

func NewTelegramListener(
	client TelegramClient,
	authenticator Authenticator,
	commandHandler CommandHandler,
) (*telegramListener, error) {
	return &telegramListener{
		client:         client,
		authenticator:  authenticator,
		commandHandler: commandHandler,
	}, nil
}

func (t *telegramListener) Name() string { return "telegram service" }

func (t *telegramListener) Run(ctx context.Context) error {
	slog.Info("starting telegram listener service")
	defer slog.Info("stopped telegram listener service")

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-t.client.GetUpdates():
			go t.processUpdate(update)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (t *telegramListener) processUpdate(update tgbotapi.Update) {
	slog.Info("Received update", "update", update)

	if update.Message != nil && !t.authenticator.IsAuthorized(update.Message.From.ID) {
		t.respondUnauthorized(update.Message.Chat.ID, update.Message.From.ID)
		return
	} else if update.CallbackQuery != nil && !t.authenticator.IsAuthorized(update.CallbackQuery.From.ID) {
		t.respondUnauthorized(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.From.ID)
		return
	}
	t.commandHandler.Handle(update)
}

func (t *telegramListener) respondUnauthorized(chatID int64, userID int64) {
	t.client.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   fmt.Sprintf("User ID %d is not authorized to use this bot.", userID),
	})
}
