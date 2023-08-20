package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/caarlos0/env/v9"
	"github.com/digitalocean/godo"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	openai "github.com/sashabaranov/go-openai"
	"golang.org/x/exp/slices"
)

type Config struct {
	GptToken                  string  `env:"GPT_TOKEN"`
	TelegramBotToken          string  `env:"TELEGRAM_BOT_TOKEN"`
	TelegramAuthorizedUserIDs []int64 `env:"TELEGRAM_AUTHORIZED_USER_IDS" envSeparator:" "`
	DigitalOceanAccessToken   string  `env:"DIGITALOCEAN_ACCESS_TOKEN"`
}

type Command struct {
	MessageID int
	ChatID    int64
	FromID    int64
	FromUser  string
	Text      string
	FileID    string
}

func main() {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		slog.Error("parsing env config", "err", err)
		os.Exit(1)
	}

	slog.Info("config", "authorized users", cfg.TelegramAuthorizedUserIDs)

	var err error
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		slog.Error("creating telegram bot", "err", err)
		os.Exit(1)
	}

	bot.Debug = true

	slog.Info("authorized on telegram", "account", bot.Self.UserName)

	executor := Executor{
		authorizedUserIDs: cfg.TelegramAuthorizedUserIDs,
		tg:                &TelegramSender{bot},
		gpt: &GptClient{
			client:       openai.NewClient(cfg.GptToken),
			chatMessages: make(map[int64][]openai.ChatCompletionMessage),
		},
		do: &DigitalOceanClient{
			client: godo.NewFromToken(cfg.DigitalOceanAccessToken),
		},
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	for update := range bot.GetUpdatesChan(u) {
		if update.CallbackQuery != nil {
			executor.HandleTextCommand(&Command{
				MessageID: update.CallbackQuery.Message.ReplyToMessage.MessageID,
				ChatID:    update.CallbackQuery.Message.Chat.ID,
				FromID:    update.CallbackQuery.From.ID,
				FromUser:  update.CallbackQuery.From.UserName,
				Text:      update.CallbackQuery.Data,
			})
		}
		if update.Message != nil {
			if update.Message.Text != "" {
				executor.HandleTextCommand(&Command{
					MessageID: update.Message.MessageID,
					ChatID:    update.Message.Chat.ID,
					FromID:    update.Message.From.ID,
					FromUser:  update.Message.From.UserName,
					Text:      update.Message.Text,
				})
			}

			if update.Message.Voice != nil || update.Message.Audio != nil {
				if update.Message.Voice != nil {
					executor.HandleAudioCommand(&Command{
						MessageID: update.Message.MessageID,
						ChatID:    update.Message.Chat.ID,
						FromID:    update.Message.From.ID,
						FromUser:  update.Message.From.UserName,
						FileID:    update.Message.Voice.FileID,
					})
				}
			}

		}
	}
}

type Executor struct {
	authorizedUserIDs []int64
	tg                *TelegramSender
	gpt               *GptClient
	do                *DigitalOceanClient
}

func (e *Executor) HandleAudioCommand(c *Command) {
	// Authorization check
	if !slices.Contains(e.authorizedUserIDs, c.FromID) {
		e.tg.SendMessage(c, fmt.Sprintf("user ID %d not authorized to use this bot", c.FromID))
	}

	filePath, err := downloadFile(e.tg.bot, c.FileID)
	if err != nil {
		e.tg.SendMessage(c, fmt.Sprintf("Failed to download file: %v", err))
		return
	}

	text, err := e.gpt.SpeechToText(filePath)
	if err != nil {
		e.tg.SendMessage(c, fmt.Sprintf("Failed to transcript audio file: %v", err))
		return
	}

	slog.Info("transcript received", "text", text)

	c.Text = text
	e.HandleTextCommand(c)
}

func (e *Executor) HandleTextCommand(c *Command) {
	// Authorization check
	if !slices.Contains(e.authorizedUserIDs, c.FromID) {
		e.tg.SendMessage(c, fmt.Sprintf("user ID %d not authorized to use this bot", c.FromID))
	}

	switch {
	case strings.HasPrefix(c.Text, "/new_chat"):
		e.gpt.ClearHistoryInChat(c.ChatID)
		e.tg.SendMessage(c, fmt.Sprintf("New chat created."))
	case strings.HasPrefix(c.Text, "/balance"):
		bill, err := e.do.GetBalanceMessage()
		if err != nil {
			e.tg.SendMessage(c, fmt.Sprintf("Failed to fetch balance for DigitalOcean: %v", err))
			return
		}
		e.tg.SendMessage(c, bill)
	case strings.HasPrefix(strings.ToLower(c.Text), "нарисуй") || strings.Contains(strings.ToLower(c.Text), "рисуй"):
		imgBytes, err := e.gpt.GenerateImage(c.Text)
		if err != nil {
			e.tg.SendMessage(c, fmt.Sprintf("Failed to generate image: %v", err))
			return
		}

		e.tg.SendImage(c, imgBytes)
	default:
		msg, err := e.gpt.GenerateMessageInChat(c.Text, c.ChatID)
		if err != nil {
			e.tg.SendMessage(c, fmt.Sprintf("Failed to get response: %v", err))
			return
		}
		e.tg.SendMessage(c, msg)
	}
}

type TelegramSender struct {
	bot *tgbotapi.BotAPI
}

func (s *TelegramSender) SendMessage(c *Command, text string) {
	msg := tgbotapi.NewMessage(c.ChatID, text)
	msg.ReplyToMessageID = c.MessageID
	if _, err := s.bot.Send(msg); err != nil {
		slog.Error("sending message to telegram", "err", err)
	}
}

func (s *TelegramSender) SendImage(m *Command, bytes []byte) {
	fileBytes := tgbotapi.FileBytes{
		Bytes: bytes,
	}
	msg := tgbotapi.NewPhoto(m.ChatID, fileBytes)
	msg.ReplyToMessageID = m.MessageID

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Еще", m.Text),
		),
	)
	msg.ReplyMarkup = keyboard

	if _, err := s.bot.Send(msg); err != nil {
		slog.Error("sending image to telegram", "err", err)
	}
}

type GptClient struct {
	client       *openai.Client
	chatMessages map[int64][]openai.ChatCompletionMessage
	mu           sync.Mutex
}

func (g *GptClient) SpeechToText(filePath string) (string, error) {
	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: filePath,
	}
	resp, err := g.client.CreateTranscription(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("creating transcription: %v", err)
	}

	return resp.Text, nil
}

func (g *GptClient) GenerateImage(prompt string) ([]byte, error) {
	req := openai.ImageRequest{
		Prompt:         prompt,
		Size:           openai.CreateImageSize512x512,
		ResponseFormat: openai.CreateImageResponseFormatB64JSON,
		N:              1,
	}

	resp, err := g.client.CreateImage(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("creating image: %v", err)
	}

	imgBytes, err := base64.StdEncoding.DecodeString(resp.Data[0].B64JSON)
	if err != nil {
		return nil, fmt.Errorf("base64 decoding: %v", err)
	}

	return imgBytes, nil
}

func (g *GptClient) ClearHistoryInChat(chatID int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.chatMessages[chatID] = nil
}

func (g *GptClient) GenerateMessageInChat(prompt string, chatID int64) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.chatMessages[chatID] = append(g.chatMessages[chatID], openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	})

	resp, err := g.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4,
			Messages: g.chatMessages[chatID],
		},
	)
	if err != nil {
		return "", fmt.Errorf("chatGPT completion: %v", err)
	}

	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no completion response from chatGPT")
}

func downloadFile(api *tgbotapi.BotAPI, fileID string) (string, error) {
	file, err := getFile(api, fileID)
	if err != nil {
		return "", fmt.Errorf("error getting file: %v", err)
	}

	filePath := path.Join("app", file.FilePath)

	req, err := createRequest(file.Link(api.Token))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	data, err := downloadFileData(api, req)
	if err != nil {
		return "", fmt.Errorf("error getting file URL: %v", err)
	}

	if err := saveFile(filePath, data); err != nil {
		return "", fmt.Errorf("error saving file: %v", err)
	}

	if path.Ext(filePath) == ".ogg" || path.Ext(filePath) == ".oga" {
		nfilePath, err := ConvertAudioToMp3(filePath)
		defer func(name string) {
			_ = os.Remove(name)
		}(filePath)
		if err != nil {
			return "", fmt.Errorf("error converting file: %v", err)
		}
		return nfilePath, nil
	}

	return filePath, nil
}

func getFile(api *tgbotapi.BotAPI, fileID string) (tgbotapi.File, error) {
	return api.GetFile(tgbotapi.FileConfig{FileID: fileID})
}

func createRequest(url string) (*http.Request, error) {
	return http.NewRequest(http.MethodGet, url, nil)
}

func downloadFileData(api *tgbotapi.BotAPI, req *http.Request) ([]byte, error) {
	resp, err := api.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	return io.ReadAll(resp.Body)
}

func saveFile(filePath string, data []byte) error {
	slog.Info("saving file", "path", filePath, "dir", path.Dir(filePath))
	if err := os.MkdirAll(path.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("saving file: %v", err)
	}
	return os.WriteFile(filePath, data, 0600)
}

func ConvertAudioToMp3(filePath string) (string, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", fmt.Errorf("unable to locate `ffmpeg`: %w", err)
	}

	npath := filePath + ".mp3"

	cmd := exec.Command("ffmpeg", "-i", filePath, npath)
	b, err := cmd.CombinedOutput()

	fmt.Println(string(b))

	if err != nil {
		return npath, fmt.Errorf("ffmpeg error: %v", err)
	}

	return npath, nil
}

type DigitalOceanClient struct {
	client *godo.Client
}

func (d *DigitalOceanClient) GetBalanceMessage() (string, error) {
	b, _, err := d.client.Balance.Get(context.Background())
	if err != nil {
		return "", fmt.Errorf("fetching balance: %v", err)
	}

	res := fmt.Sprintf("Server Balance Info: \nMonth-To-Date Balance: $%v \nAccount Balance: $%v",
		b.MonthToDateBalance, b.AccountBalance)
	return res, nil
}
