package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Command interface {
	CanExecute(update *tgbotapi.Update) bool
	Execute(update *tgbotapi.Update)
}

type commandHandler struct {
	commands []Command
}

func NewCommandHandler(commands []Command) *commandHandler {
	return &commandHandler{
		commands: commands,
	}
}

func (с *commandHandler) Handle(update tgbotapi.Update) {
	for _, command := range с.commands {
		if command.CanExecute(&update) {
			command.Execute(&update)
			return
		}
	}
}
