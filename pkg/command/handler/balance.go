package handler

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type DigitalOceanBalanceProvider interface {
	GetBalanceMessage(ctx context.Context) (string, error)
}

type balance struct {
	provider DigitalOceanBalanceProvider
	outCh    chan<- domain.Message
}

func NewBalance(
	provider DigitalOceanBalanceProvider,
	outCh chan<- domain.Message,
) *balance {
	return &balance{
		provider: provider,
		outCh:    outCh,
	}
}

func (b *balance) CanHandle(update *tgbotapi.Update) bool {
	return update.Message != nil && strings.HasPrefix(update.Message.Text, "/balance")
}

func (b *balance) Handle(update *tgbotapi.Update) {
	response, err := b.provider.GetBalanceMessage(context.TODO())
	if err != nil {
		response = fmt.Sprintf("Failed to fetch DigitalOcean balance: %v", err)
	}

	b.outCh <- &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          response,
	}
}
