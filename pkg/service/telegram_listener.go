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

	workerPool := make(chan struct{}, 10) // Max 10 concurrent workers

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-t.client.GetUpdates():
			workerPool <- struct{}{}
			go func(update tgbotapi.Update) {
				defer func() { <-workerPool }()
				t.processUpdate(update)
			}(update)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (t *telegramListener) processUpdate(update tgbotapi.Update) {
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

	t.commandHandler.Handle(update)
}

func (t *telegramListener) respondUnauthorized(chatID int64, userID int64) {
	slog.Warn("Unauthorized access attempt", "userID", userID, "chatID", chatID)

	t.client.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   fmt.Sprintf("User ID %d is not authorized", userID),
	})
}
