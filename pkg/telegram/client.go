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
	maxRunes = 4096

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

func (c *client) SendTextMessage(msg domain.TextMessage) {
	// convert to HTML
	text := render.ToHTML(msg.Text)

	// check the rune count, telegram is limited to 4096 chars per message
	msgRuneCount := utf8.RuneCountInString(text)

	for msgRuneCount > maxRunes {
		stop := maxRunes

		// Find the last <pre> tag or newline before the stop index
		lastPreTag := strings.LastIndex(text[:stop], "<pre>")
		lastNewline := strings.LastIndex(text[:stop], "\n")

		// Choose the appropriate stop point
		if lastPreTag != -1 && lastPreTag < stop {
			stop = lastPreTag
		} else if lastNewline != -1 && lastNewline < stop {
			stop = lastNewline
		}

		// Send the current chunk
		c.send(msg.ChatID, text[:stop])

		text = text[stop:]
		msgRuneCount = utf8.RuneCountInString(text)
	}

	c.send(msg.ChatID, text)
}

func (c *client) send(chatID int64, text string) {
	m := tgbotapi.NewMessage(chatID, text)
	m.ParseMode = tgbotapi.ModeHTML
	m.DisableWebPagePreview = true

	if _, err := c.bot.Send(m); err != nil {
		c.handleError(chatID, err)
	}
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
