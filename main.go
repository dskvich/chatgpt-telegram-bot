package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/franciscoescher/goopenai"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	authorizedUserIDs []int64
    clientSessions sync.Map
)

func main() {
	authorizedUserIDs = parseAuthorizedUserIDs(os.Getenv("TELEGRAM_AUTHORIZED_USER_IDS"))

	lambda.Start(Handler)
}

func checkInternetAccess() bool {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://www.google.com")
	if err != nil {
		return false
	}

	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
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


func Handler(ctx context.Context, update tgbotapi.Update) error {
	if checkInternetAccess() {
		log.Println("Internet access is available")
	} else {
		log.Println("No internet access")
	}

	log.Printf("Authorized IDs parsed: %+v\n", authorizedUserIDs)

	bot, err := tgbotapi.NewBotAPI("TELEGRAM_BOT_TOKEN")
	if err != nil {
		return fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	if update.Message != nil {
		if !isAuthorizedUser(update.Message.From.ID) {
			log.Printf("Unauthorized user: %d", update.Message.From.ID)
			return nil
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		oAPIClient, err := getOrCreateChatGPTClient(update.Message.From.ID)
		if err != nil {
			log.Println(err)
			return fmt.Errorf("failed to get or create ChatGPT client: %w", err)
		}

		response, err := sendToChatGPT(ctx, oAPIClient, update.Message.Text)
		if err != nil {
			log.Println(err)
			return err
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
		msg.ReplyToMessageID = update.Message.MessageID

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("failed to send message via Telegram bot: %w", err)
		}
	}

	return nil
}

func isAuthorizedUser(userID int64) bool {
	for _, id := range authorizedUserIDs {
		if id == userID {
			return true
		}
	}

	return false
}

func getOrCreateChatGPTClient(userID int64) (*goopenai.Client, error) {
	if val, ok := clientSessions.Load(userID); ok {
		if client, ok := val.(*goopenai.Client); ok {
			return client, nil
		}
	}

	client := goopenai.NewClient(os.Getenv("GPT_TOKEN"), "")
	clientSessions.Store(userID, client)

	return client, nil
}

func sendToChatGPT(ctx context.Context, oAPIClient *goopenai.Client, message string) (string, error) {
	r := goopenai.CreateCompletionsRequest{
		Model: "gpt-3.5-turbo",
		Messages: []goopenai.Message{
			{
				Role:    "user",
				Content: message,
			},
		},
	}

	completions, err := oAPIClient.CreateCompletions(ctx, r)
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