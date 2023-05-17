package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/franciscoescher/goopenai"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/exp/slices"
)

var allowedIds  []int64

func main() {
	for _, s := range strings.Split(os.Getenv("TELEGRAM_ALLOWED_IDS"), ",") {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			log.Printf("Error converting string '%s' to int64: %v\n", s, err)
			continue
		}
		allowedIds = append(allowedIds, id)
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	oAPIClient := goopenai.NewClient(os.Getenv("GPT_TOKEN"), "")

	for update := range updates {
		if update.Message != nil { // If we got a message
			if !isAllowed(update.Message.From.ID) {
				continue
			}
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			r := goopenai.CreateCompletionsRequest{
				Model: "gpt-3.5-turbo",
				Messages: []goopenai.Message{
					{
						Role:    "user",
						Content: update.Message.Text,
					},
				},
			}

			completions, err := oAPIClient.CreateCompletions(context.Background(), r)
			if err != nil {
				log.Println(err)
				continue
			}

			if completions.Error.Message != "" {
				log.Println(completions.Error)
				continue
			}

			if len(completions.Choices) > 0 && completions.Choices[0].Message.Content != "" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, completions.Choices[0].Message.Content)
				msg.ReplyToMessageID = update.Message.MessageID

				if _, err := bot.Send(msg); err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func isAllowed(id int64) bool {
	return slices.Contains(allowedIds, id)
}
