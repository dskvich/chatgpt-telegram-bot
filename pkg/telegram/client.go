package telegram

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/render"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
)

const maxTelegramMessageLength = 4096

type client struct {
	token     string
	bot       *tgbotapi.BotAPI
	updatesCh tgbotapi.UpdatesChannel

	// TODO: move this in service or something
	imageStyles map[string]string
}

func NewClient(token string, imageStyles map[string]string) (*client, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating bot api instance: %w", err)
	}

	slog.Info("authorized on telegram", "account", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	return &client{
		token:       token,
		bot:         bot,
		updatesCh:   bot.GetUpdatesChan(u),
		imageStyles: imageStyles,
	}, nil
}

func (c *client) GetUpdates() tgbotapi.UpdatesChannel {
	return c.updatesCh
}

func (c *client) SendResponse(ctx context.Context, chatID int64, response *domain.Response) {
	if response == nil {
		return
	}

	if response.Text != "" {
		if err := c.sendText(ctx, chatID, response.Text); err != nil {
			c.handleError(ctx, chatID, err)
		}
	}

	if response.Image != nil {
		if err := c.sendImage(ctx, chatID, response.Image); err != nil {
			c.handleError(ctx, chatID, err)
		}
	}
}

func (c *client) SendError(ctx context.Context, chatID int64, err error) {
	slog.ErrorContext(ctx, "error occurred", "chatID", chatID, logger.Err(err))

	if err := c.sendText(ctx, chatID, fmt.Sprintf("❌ %s", err.Error())); err != nil {
		c.handleError(ctx, chatID, err)
	}
}

func (c *client) sendText(ctx context.Context, chatID int64, text string) error {
	htmlText := render.ToHTML(text)

	for htmlText != "" {
		if utf8.RuneCountInString(htmlText) <= maxTelegramMessageLength {
			if err := c.send(chatID, htmlText); err != nil {
				return err
			}
			return nil
		}

		cutIndex := c.findCutIndex(htmlText, maxTelegramMessageLength)
		if err := c.send(chatID, htmlText); err != nil {
			return err
		}
		htmlText = htmlText[cutIndex:]

		// 1 message per second
		time.Sleep(time.Second)
	}

	return nil
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

func (c *client) sendImage(ctx context.Context, chatID int64, image *domain.Image) error {
	data := fmt.Sprintf("%s%d", domain.GenImageCallbackPrefix, image.ID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Еще", data),
		),
	)

	m := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Bytes: image.Data})
	m.ReplyMarkup = keyboard

	if _, err := c.bot.Send(m); err != nil {
		return err
	}

	return nil
}

func (c *client) SendKeyboard(ctx context.Context, chatID int64, options map[string]string, title string) {
	var row []tgbotapi.InlineKeyboardButton

	for label, callback := range options {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, callback))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)

	message := tgbotapi.NewMessage(chatID, title)
	message.ReplyMarkup = keyboard

	if _, err := c.bot.Send(message); err != nil {
		c.handleError(ctx, chatID, err)
	}
}

func (c *client) SendImageStyleMessage(ctx context.Context, msg domain.TextMessage) {
	var keyboardButtons []tgbotapi.InlineKeyboardButton
	for key, label := range c.imageStyles {
		callbackData := "image_style_" + key // Add the prefix
		keyboardButtons = append(keyboardButtons, tgbotapi.NewInlineKeyboardButtonData(label, callbackData))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			keyboardButtons[0],
			keyboardButtons[1],
		),
	)

	m := tgbotapi.NewMessage(msg.ChatID, "Выберите стиль генерации изображений: ")
	m.ReplyMarkup = keyboard

	if _, err := c.bot.Send(m); err != nil {
		c.handleError(ctx, msg.ChatID, err)
	}
}

func (c *client) AcknowledgeCallback(ctx context.Context, callbackQueryID string) {
	m := tgbotapi.NewCallback(callbackQueryID, "")

	_, _ = c.bot.Send(m)
}

func (c *client) handleError(ctx context.Context, chatID int64, err error) {
	slog.ErrorContext(ctx, "Error during sending message", logger.Err(err))

	m := tgbotapi.NewMessage(chatID, "❌ Не удалось доставить ответ")

	_, _ = c.bot.Send(m)
}

func (c *client) DownloadFile(ctx context.Context, fileID string) ([]byte, error) {
	file, err := c.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("getting file: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, file.Link(c.token), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.bot.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return bytes, nil
}
