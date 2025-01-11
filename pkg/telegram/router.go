package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MessageHandler interface {
	CanHandleMessage(*tgbotapi.Update) bool
	HandleMessage(*tgbotapi.Update)
}

type CallbackHandler interface {
	CanHandleCallback(*tgbotapi.Update) bool
	HandleCallback(*tgbotapi.Update)
}

type router struct {
	handlers []any
}

func NewRouter(handlers []any) *router {
	return &router{
		handlers: handlers,
	}
}

func (r *router) Handle(update tgbotapi.Update) {
	for _, h := range r.handlers {
		if cmd, ok := h.(MessageHandler); ok && cmd.CanHandleMessage(&update) {
			cmd.HandleMessage(&update)
			return
		}
		if cb, ok := h.(CallbackHandler); ok && cb.CanHandleCallback(&update) {
			cb.HandleCallback(&update)
			return
		}
	}
}
