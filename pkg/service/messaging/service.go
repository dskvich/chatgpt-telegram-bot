package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
)

type Authenticator interface {
	IsAuthorized(userID int64) bool
}

type TelegramClient interface {
	GetUpdates() tgbotapi.UpdatesChannel
	Send(message domain.Message) error
}

type CommandHandler interface {
	Handle(update tgbotapi.Update)
}

type service struct {
	telegramClient   TelegramClient
	authenticator    Authenticator
	commandHandler   CommandHandler
	outgoingMessages chan domain.Message
}

func NewService(
	telegramClient TelegramClient,
	authenticator Authenticator,
	commandHandler CommandHandler,
	outgoingMessages chan domain.Message,
) (*service, error) {
	return &service{
		telegramClient:   telegramClient,
		authenticator:    authenticator,
		commandHandler:   commandHandler,
		outgoingMessages: outgoingMessages,
	}, nil
}

func (s *service) Name() string { return "telegram bot" }

func (s *service) Run(ctx context.Context) error {
	slog.Info("starting messaging service")
	defer slog.Info("stopped messaging service")

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-s.telegramClient.GetUpdates():
			go s.processUpdate(update)
		case message := <-s.outgoingMessages:
			go s.sendMessage(message)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (s *service) processUpdate(update tgbotapi.Update) {
	slog.Info("Received update", "update", update)

	if update.Message != nil && !s.authenticator.IsAuthorized(update.Message.From.ID) {
		s.respondUnauthorized(update.Message.Chat.ID, update.Message.MessageID, update.Message.From.ID)
		return
	} else if update.CallbackQuery != nil && !s.authenticator.IsAuthorized(update.CallbackQuery.From.ID) {
		s.respondUnauthorized(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, update.CallbackQuery.From.ID)
		return
	}
	s.commandHandler.Handle(update)
}

func (s *service) sendMessage(message domain.Message) {
	if err := s.telegramClient.Send(message); err != nil {
		slog.Error("sending message", "message", message, logger.Err(err))
	}
}

func (s *service) respondUnauthorized(chatID int64, replyToMessageID int, userID int64) {
	message := &domain.TextMessage{
		ChatID:           chatID,
		ReplyToMessageID: replyToMessageID,
		Content:          fmt.Sprintf("User ID %d is not authorized to use this bot.", userID),
	}
	s.outgoingMessages <- message
}
