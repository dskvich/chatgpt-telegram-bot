package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/franciscoescher/goopenai"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	authorizedUserIDs []int64
	client            *goopenai.Client
)

func main() {
	authorizedUserIDs = parseAuthorizedUserIDs(os.Getenv("TELEGRAM_AUTHORIZED_USER_IDS"))
	client = goopenai.NewClient(os.Getenv("GPT_TOKEN"), "")

	lambda.Start(Handler)
}

func parseAuthorizedUserIDs(str string) []int64 {
	var res []int64

	ids := strings.Split(str, ",")
	for _, id := range ids {
		userID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			log.Printf("Error converting string '%s' to int64: %v\n", id, err)
			continue
		}
		res = append(res, userID)
	}

	return res
}

func Handler(ctx context.Context, update tgbotapi.Update) {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		log.Fatalf("failed to create Telegram bot: %w", err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	if update.Message != nil {
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		switch {
		case strings.HasPrefix(update.Message.Text, "/userid"):
			handleUserIDCommand(ctx, bot, update)
		default:
			handleChatMessage(ctx, bot, update)
		}
	}
}

func handleUserIDCommand(_ context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	response := fmt.Sprintf("Your user ID is %d", update.Message.From.ID)
	sendTelegramMessage(bot, update.Message.Chat.ID, response, update.Message.MessageID)
}

func sendTelegramMessage(bot *tgbotapi.BotAPI, chatID int64, text string, replyToMessageID int) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = replyToMessageID
	if _, err := bot.Send(msg); err != nil {
		log.Fatalf("failed to send message via Telegram bot: %w", err)
	}
}

func handleChatMessage(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if !isAuthorizedUser(update.Message.From.ID) {
		log.Fatalf("Unauthorized user: %d", update.Message.From.ID)
	}

	response, err := sendToChatGPT(ctx, update.Message.Text)
	if err != nil {
		log.Fatalf("failed to send message to ChatGPT: %w", err)
	}

	sendTelegramMessage(bot, update.Message.Chat.ID, response, update.Message.MessageID)
}

func isAuthorizedUser(userID int64) bool {
	for _, id := range authorizedUserIDs {
		if id == userID {
			return true
		}
	}

	return false
}

func sendToChatGPT(ctx context.Context, message string) (string, error) {
	r := goopenai.CreateCompletionsRequest{
		Model: "gpt-3.5-turbo",
		Messages: []goopenai.Message{
			{
				Role:    "user",
				Content: message,
			},
		},
	}

	completions, err := client.CreateCompletions(ctx, r)
	if err != nil {
		return "", err
	}

	if completions.Error.Message != "" {
		return "", fmt.Errorf("chatGPT error: %s", completions.Error.Message)
	}

	if len(completions.Choices) > 0 && completions.Choices[0].Message.Content != "" {
		return completions.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no completion response from ChatGPT")
}
