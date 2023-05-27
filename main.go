package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/franciscoescher/goopenai"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	imageGenerationEndpoint = "https://api.openai.com/v1/images/generations"
	imageSize               = "1024x1024"
	imageResponseFormat     = "b64_json"
)

var (
	authorizedUserIDs []int64
	client            *goopenai.Client
	imageCommandRegexp = regexp.MustCompile(`/image(-*\d*)\s+(.*)`)
	httpClient = http.Client{}
)

func main() {
	authorizedUserIDs = parseAuthorizedUserIDs(os.Getenv("TELEGRAM_AUTHORIZED_USER_IDS"))
	client = goopenai.NewClient(os.Getenv("GPT_TOKEN"), "")

	port := os.Getenv("PORT")
	go runServer(port)

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		log.Fatalf("failed to create Telegram bot: %v", err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			if strings.HasPrefix(update.Message.Text, "/userid") {
				if err := handleUserIDCommand(bot, update); err != nil {
					log.Println(err)
					continue
				}
			}

			if strings.HasPrefix(update.Message.Text, "/image") {
				if err := handleImageCommand(bot, update); err != nil {
					log.Println(err)
					continue
				}
			} else {
				if err := handleChatMessage(bot, update); err != nil {
					log.Println(err)
					continue
				}
			}
		}
	}
}

func runServer(port string) {
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func parseAuthorizedUserIDs(str string) []int64 {
	if str == "" {
		return nil
	}

	var res []int64

	ids := strings.Split(str, " ")
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

func handleUserIDCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	response := fmt.Sprintf("Your user ID is %d", update.Message.From.ID)
	return sendTelegramMessage(bot, update.Message.Chat.ID, response, update.Message.MessageID)
}

func sendTelegramMessage(bot *tgbotapi.BotAPI, chatID int64, text string, replyToMessageID int) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = replyToMessageID
	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send message via Telegram bot: %w", err)
	}
	return nil
}

func handleImageCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if len(authorizedUserIDs) > 0 && !isAuthorizedUser(update.Message.From.ID) {
		return fmt.Errorf("unauthorized user: %d", update.Message.From.ID)
	}

	num, prompt := parseImageCommand(update.Message.Text)
	imageBinarySlice, err := generateImages(num, prompt)
	if err != nil {
		return sendTelegramMessage(bot, update.Message.Chat.ID, err.Error(), update.Message.MessageID)
	}

	for _, imageBinary := range imageBinarySlice {
		sendTelegramImage(bot, update.Message.Chat.ID, update.Message.MessageID, imageBinary)
	}

	return nil
}

func parseImageCommand(command string) (int, string) {
	match := imageCommandRegexp.FindStringSubmatch(command)
	if len(match) > 2 {
		numOfImagesStr := match[1]
		numOfImages, _ := strconv.Atoi(numOfImagesStr)
		if numOfImages < 2 {
			numOfImages = 1
		}
		prompt := match[2]
		return numOfImages, prompt
	}

	return 1, command
}

func sendTelegramImage(bot *tgbotapi.BotAPI, chatID int64, replyToMessageID int, imageData []byte) error {
	fileBytes := tgbotapi.FileBytes{
		Bytes: imageData,
	}
	msg := tgbotapi.NewPhoto(chatID, fileBytes)
	msg.ReplyToMessageID = replyToMessageID
	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send image via Telegram bot: %w", err)
	}
	return nil
}

func handleChatMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if len(authorizedUserIDs) > 0 && !isAuthorizedUser(update.Message.From.ID) {
		return fmt.Errorf("unauthorized user: %d", update.Message.From.ID)
	}

	response, err := sendToChatGPT(update.Message.Text)
	if err != nil {
		return sendTelegramMessage(bot, update.Message.Chat.ID, err.Error(), update.Message.MessageID)
	}

	return sendTelegramMessage(bot, update.Message.Chat.ID, response, update.Message.MessageID)
}

type CreateImageRequest struct {
	Prompt         string `json:"prompt"`
	Number         int    `json:"n"`
	Size           string `json:"size"`
	ResponseFormat string `json:"response_format"`
}

type CreateImageResponse struct {
	Created int `json:"created"`
	Data    []struct {
		B64JSON []byte `json:"b64_json"`
	} `json:"data"`
}

func generateImages(num int, prompt string) ([][]byte, error) {
	request := CreateImageRequest{
		Prompt:         prompt,
		Number:         num,
		Size:           imageSize,
		ResponseFormat: imageResponseFormat,
	}

	bodyBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	log.Printf("create image request: %+v", string(bodyBytes))

	req, err := http.NewRequest(http.MethodPost, imageGenerationEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Fatal(err)
	}

	bearer := "Bearer " + os.Getenv("GPT_TOKEN")
	req.Header.Add("Authorization", bearer)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid status code: %d, body: %+v", resp.StatusCode, string(responseData))
	}

	var response CreateImageResponse
	if err = json.Unmarshal(responseData, &response); err != nil {
		return nil, err
	}

	var res [][]byte
	for _, b := range response.Data {
		res = append(res, b.B64JSON)
	}
	return res, nil
}

func isAuthorizedUser(userID int64) bool {
	for _, id := range authorizedUserIDs {
		if id == userID {
			return true
		}
	}

	return false
}

func sendToChatGPT(message string) (string, error) {
	r := goopenai.CreateCompletionsRequest{
		Model: "gpt-3.5-turbo",
		Messages: []goopenai.Message{
			{
				Role:    "user",
				Content: message,
			},
		},
	}

	completions, err := client.CreateCompletions(context.Background(), r)
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
