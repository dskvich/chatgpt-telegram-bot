package command

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type Handler interface {
	CanHandle(update *tgbotapi.Update) bool
	Handle(update *tgbotapi.Update) domain.Message
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

func (d *dispatcher) Dispatch(update tgbotapi.Update) domain.Message {
	for _, handler := range d.handlers {
		if handler.CanHandle(&update) {
			return handler.Handle(&update)
		}
	}

	return d.defaultHandler.Handle(&update) // ?
}
