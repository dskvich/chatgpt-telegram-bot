package telegram

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/render"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
)

const (
	maxTelegramMessageLength = 4096

	responseDeliveryFailedMessage = "Не удалось доставить ответ"
)

type client struct {
	token     string
	bot       *tgbotapi.BotAPI
	updatesCh tgbotapi.UpdatesChannel
}

func NewClient(token string) (*client, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating bot api instance: %w", err)
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

func (c *client) SendTextMessage(msg domain.TextMessage) {
	text := render.ToHTML(msg.Text)

	for len(text) > 0 {
		if utf8.RuneCountInString(text) <= maxTelegramMessageLength {
			if err := c.send(msg.ChatID, text); err != nil {
				c.handleError(msg.ChatID, err)
			}
			break
		}

		cutIndex := c.findCutIndex(text, maxTelegramMessageLength)
		if err := c.send(msg.ChatID, text); err != nil {
			c.handleError(msg.ChatID, err)
		}
		text = text[cutIndex:]
	}
}

func (c *client) findCutIndex(text string, maxLength int) int {
	lastPre := strings.LastIndex(text[:maxLength], "<pre>")
	lastNewline := strings.LastIndex(text[:maxLength], "\n")

	if lastPre > -1 {
		return lastPre
	}
	if lastNewline > -1 {
		return lastNewline
	}
	return maxLength
}

func (c *client) send(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true

	_, err := c.bot.Send(msg)
	return err
}

func (c *client) SendImageMessage(msg domain.ImageMessage) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Еще", domain.RedrawCallback),
		),
	)

	m := tgbotapi.NewPhoto(msg.ChatID, tgbotapi.FileBytes{Bytes: msg.Bytes})
	m.ReplyMarkup = keyboard

	if _, err := c.bot.Send(m); err != nil {
		c.handleError(msg.ChatID, err)
	}
}

func (c *client) SendTTLMessage(msg domain.TTLMessage) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("15min", "ttl_15m"),
			tgbotapi.NewInlineKeyboardButtonData("1h", "ttl_1h"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("8h", "ttl_8h"),
			tgbotapi.NewInlineKeyboardButtonData("Disabled", "ttl_disabled"),
		),
	)

	m := tgbotapi.NewMessage(msg.ChatID, "Select TTL option:")
	m.ReplyMarkup = keyboard

	if _, err := c.bot.Send(m); err != nil {
		c.handleError(msg.ChatID, err)
	}
}

func (c *client) SendImageStyleMessage(msg domain.TextMessage) {
	var keyboardButtons []tgbotapi.InlineKeyboardButton
	for key, label := range domain.ImageStyles {
		callbackData := "image_style_" + key // Add the prefix
		keyboardButtons = append(keyboardButtons, tgbotapi.NewInlineKeyboardButtonData(label, callbackData))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			keyboardButtons[0],
			keyboardButtons[1],
		),
	)

	m := tgbotapi.NewMessage(msg.ChatID, domain.GetImageStylePrompt())
	m.ReplyMarkup = keyboard

	if _, err := c.bot.Send(m); err != nil {
		c.handleError(msg.ChatID, err)
	}
}

func (c *client) SendCallbackMessage(msg domain.CallbackMessage) {
	m := tgbotapi.NewCallback(msg.CallbackQueryID, "")

	_, _ = c.bot.Send(m)
}

func (c *client) handleError(chatID int64, err error) {
	slog.Error("sending message error", "chatID", chatID, logger.Err(err))

	m := tgbotapi.NewMessage(chatID, responseDeliveryFailedMessage)

	_, _ = c.bot.Send(m)
}

func (c *client) DownloadFile(fileID string) (string, error) {
	file, err := c.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("getting file: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, file.Link(c.token), nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.bot.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		if closeErr := Body.Close(); closeErr != nil {
			slog.Error("closing body", logger.Err(closeErr))
		}
	}(resp.Body)

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	filePath := path.Join("app", file.FilePath)
	if err := os.MkdirAll(path.Dir(filePath), 0o755); err != nil {
		return "", fmt.Errorf("creating directories for '%s': %w", filePath, err)
	}

	if err := os.WriteFile(filePath, bytes, 0o600); err != nil {
		return "", fmt.Errorf("saving file: %w", err)
	}

	return filePath, nil
}

func (c *client) GetFile(fileID string) (string, error) {
	file, err := c.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("getting file: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, file.Link(c.token), nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.bot.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		if closeErr := Body.Close(); closeErr != nil {
			slog.Error("closing body", logger.Err(closeErr))
		}
	}(resp.Body)

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(bytes)
	return base64Image, nil
}
