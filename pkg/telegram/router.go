package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Command interface {
	IsCommand(*tgbotapi.Update) bool
	HandleCommand(*tgbotapi.Update)
}

type Callback interface {
	IsCallback(*tgbotapi.Update) bool
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
		if cmd, ok := h.(Command); ok && cmd.IsCommand(&update) {
			cmd.HandleCommand(&update)
			return
		}
		if cb, ok := h.(Callback); ok && cb.IsCallback(&update) {
			cb.HandleCallback(&update)
			return
		}
	}
}
