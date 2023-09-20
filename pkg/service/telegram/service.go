package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/logger"
)

type Authenticator interface {
	IsAuthorized(userID int64) bool
}

type Bot interface {
	GetUpdates() tgbotapi.UpdatesChannel
	Send(message domain.Message) error
}

type CommandDispatcher interface {
	Dispatch(update tgbotapi.Update) domain.Message
}

type service struct {
	bot               Bot
	authenticator     Authenticator
	commandDispatcher CommandDispatcher
	messages          chan domain.Message
}

func NewService(
	bot Bot,
	authenticator Authenticator,
	commandDispatcher CommandDispatcher,
	messages chan domain.Message,
) (*service, error) {
	return &service{
		bot:               bot,
		authenticator:     authenticator,
		commandDispatcher: commandDispatcher,
		messages:          messages,
	}, nil
}

func (s *service) Name() string { return "telegram bot" }

func (s *service) Run(ctx context.Context) error {
	slog.Info("starting telegram bot service")
	defer slog.Info("stopped telegram bot service")

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-s.bot.GetUpdates():
			go s.handleUpdate(update)
		case message := <-s.messages:
			go s.handleMessage(message)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (s *service) handleUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		slog.Info("update message received", "chat", update.Message.Chat.ID, "user", update.Message.From, "text", update.Message.Text)

		if !s.authenticator.IsAuthorized(update.Message.From.ID) {
			s.messages <- &domain.TextMessage{
				ChatID:           update.Message.Chat.ID,
				ReplyToMessageID: update.Message.MessageID,
				Content:          fmt.Sprintf("User ID %d not authorized to use this bot.", update.Message.From.ID),
			}
			return
		}

		s.messages <- s.commandDispatcher.Dispatch(update)
	} else if update.CallbackQuery != nil {
		slog.Info("callback received", "data", update.CallbackQuery.Data, "user", update.CallbackQuery.Message.From, "message", update.CallbackQuery.Message)

		if !s.authenticator.IsAuthorized(update.CallbackQuery.Message.Chat.ID) {
			s.messages <- &domain.TextMessage{
				ChatID:           update.CallbackQuery.Message.Chat.ID,
				ReplyToMessageID: update.CallbackQuery.Message.MessageID,
				Content:          fmt.Sprintf("User ID %d not authorized to use this bot.", update.CallbackQuery.Message.From.ID),
			}
			return
		}

		s.messages <- s.commandDispatcher.Dispatch(update)
	}
}

func (s *service) handleMessage(message domain.Message) {
	if err := s.bot.Send(message); err != nil {
		slog.Error("sending message", logger.Err(err))
	}
}
