package telegram

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler interface {
	CanHandle(update *tgbotapi.Update) bool
	Handle(ctx context.Context, update *tgbotapi.Update)
}

type Registry struct {
	handlers []Handler
}

func NewRegistry(handlers ...Handler) *Registry {
	return &Registry{handlers: handlers}
}

func (r *Registry) HandleUpdate(ctx context.Context, update *tgbotapi.Update) {
	for _, handler := range r.handlers {
		if handler.CanHandle(update) {
			slog.InfoContext(ctx, "Calling handler", "handler", fmt.Sprintf("%T", handler))

			handler.Handle(ctx, update)
			return
		}
	}
	slog.WarnContext(ctx, "No handler found for update")
}
