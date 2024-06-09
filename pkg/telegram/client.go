package telegram

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
)

type client struct {
	token     string
	bot       *tgbotapi.BotAPI
	updatesCh tgbotapi.UpdatesChannel
}

func NewClient(token string) (*client, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating bot api instance: %v", err)
	}

	slog.Info("authorized on telegram", "account", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	return &client{
		token:     token,
		bot:       bot,
		updatesCh: bot.GetUpdatesChan(u),
	}, nil
}

func (c *client) GetUpdates() tgbotapi.UpdatesChannel {
	return c.updatesCh
}

func (c *client) Send(message domain.Message) error {
	chatMessage := message.ToChatMessage()
	if _, err := c.bot.Send(chatMessage); err != nil {
		return c.handleError(chatMessage, err)
	}
	return nil
}

func (c *client) handleError(chatMessage tgbotapi.Chattable, err error) error {
	messageConfig, ok := chatMessage.(tgbotapi.MessageConfig)
	if !ok {
		slog.Error("type assertion failed", "expectedType", "tgbotapi.MessageConfig", "actualType", fmt.Sprintf("%T", chatMessage))
		return fmt.Errorf("unexpected type %T for chatMessage", chatMessage)
	}

	errorMessage := domain.TextMessage{
		ChatID:           messageConfig.ChatID,
		ReplyToMessageID: messageConfig.ReplyToMessageID,
		Content:          "Не удалось доставить ответ",
	}
	if _, err := c.bot.Send(errorMessage.ToChatMessage()); err != nil {
		return fmt.Errorf("sending failure notification: %v", err)
	}

	return fmt.Errorf("sending message: %v", err)
}
func (c *client) DownloadFile(fileID string) (string, error) {
	file, err := c.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("getting file: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, file.Link(c.token), nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %v", err)
	}

	resp, err := c.bot.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		if closeErr := Body.Close(); closeErr != nil {
			slog.Error("closing body", logger.Err(closeErr))
		}
	}(resp.Body)

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %v", err)
	}

	filePath := path.Join("app", file.FilePath)
	if err := os.MkdirAll(path.Dir(filePath), 0755); err != nil {
		return "", fmt.Errorf("creating directories for '%s': %v", filePath, err)
	}

	if err := os.WriteFile(filePath, bytes, 0600); err != nil {
		return "", fmt.Errorf("saving file: %v", err)
	}

	return filePath, nil
}

func (c *client) GetFile(fileID string) (string, error) {
	file, err := c.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("getting file: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, file.Link(c.token), nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %v", err)
	}

	resp, err := c.bot.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		if closeErr := Body.Close(); closeErr != nil {
			slog.Error("closing body", logger.Err(closeErr))
		}
	}(resp.Body)

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %v", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(bytes)
	return base64Image, nil
}
