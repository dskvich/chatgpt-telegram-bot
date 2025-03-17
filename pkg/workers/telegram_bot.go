package workers

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
)

type telegramBot struct {
	bot *bot.Bot
}

func NewTelegramBot(bot *bot.Bot) (*telegramBot, error) {
	return &telegramBot{
		bot: bot,
	}, nil
}

func (t *telegramBot) Name() string { return "telegram_bot" }

func (t *telegramBot) Start(ctx context.Context) error {
	slog.Info("Starting worker", "name", t.Name())
	defer slog.Info("Worker stopped", "name", t.Name())

	t.bot.Start(ctx)

	return nil
}
