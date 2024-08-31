package command

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type TTLSetter interface {
	SetTTL(chatID int64, ttl time.Duration)
}

type setTTL struct {
	client    TelegramClient
	ttlSetter TTLSetter
}

func NewSetTTL(
	client TelegramClient,
	ttlSetter TTLSetter,
) *setTTL {
	return &setTTL{
		client:    client,
		ttlSetter: ttlSetter,
	}
}

func (c *setTTL) IsCommand(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(strings.ToLower(u.Message.Text), "/ttl")
}

func (c *setTTL) HandleCommand(u *tgbotapi.Update) {
	c.client.SendTTLMessage(domain.TTLMessage{
		ChatID:           u.Message.Chat.ID,
		ReplyToMessageID: u.Message.MessageID,
	})
}

func (c *setTTL) IsCallback(u *tgbotapi.Update) bool {
	return u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, domain.SetChatTTLCallback)
}

func (c *setTTL) HandleCallback(u *tgbotapi.Update) {
	chatID := u.CallbackQuery.Message.Chat.ID
	messageID := u.CallbackQuery.Message.ReplyToMessage.MessageID
	callbackQueryID := u.CallbackQuery.ID

	ttl, err := c.parseTTL(u.CallbackQuery.Data)
	if err != nil {
		c.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             "Unknown TTL option selected.",
		})
		return
	}

	c.ttlSetter.SetTTL(chatID, ttl)

	c.client.SendCallbackMessage(domain.CallbackMessage{
		CallbackQueryID: callbackQueryID,
	})

	ttlText := "disabled"
	if ttl > 0 {
		ttlText = fmt.Sprintf("%v", ttl)
	}

	c.client.SendTextMessage(domain.TextMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Text:             fmt.Sprintf("Set TTL to %v", ttlText),
	})
}

func (c *setTTL) parseTTL(data string) (time.Duration, error) {
	switch data {
	case "ttl_15m":
		return 15 * time.Minute, nil
	case "ttl_1h":
		return time.Hour, nil
	case "ttl_8h":
		return 8 * time.Hour, nil
	case "ttl_disabled":
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown TTL option")
	}
}
