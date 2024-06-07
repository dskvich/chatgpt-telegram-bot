package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Command interface {
	CanExecute(update *tgbotapi.Update) bool
	Execute(update *tgbotapi.Update)
}

type commandDispatcher struct {
	commands []Command
}

func NewCommandDispatcher(commands []Command) *commandDispatcher {
	return &commandDispatcher{
		commands: commands,
	}
}

func (d *commandDispatcher) ExecuteCommands(update tgbotapi.Update) {
	for _, command := range d.commands {
		if command.CanExecute(&update) {
			command.Execute(&update)
		}
	}
}
