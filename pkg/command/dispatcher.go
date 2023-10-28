package command

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler interface {
	CanHandle(update *tgbotapi.Update) bool
	Handle(update *tgbotapi.Update)
}

type dispatcher struct {
	handlers       []Handler
	defaultHandler Handler
}

func NewDispatcher(handlers []Handler, defaultHandler Handler) *dispatcher {
	return &dispatcher{
		handlers:       handlers,
		defaultHandler: defaultHandler,
	}
}

func (d *dispatcher) Dispatch(update tgbotapi.Update) {
	for _, handler := range d.handlers {
		if handler.CanHandle(&update) {
			handler.Handle(&update)
			return
		}
	}

	d.defaultHandler.Handle(&update)
}
