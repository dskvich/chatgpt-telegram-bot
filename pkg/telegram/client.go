package telegram

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/render"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/ratelimit"
)

const maxTelegramMessageLength = 4096

type client struct {
	token       string
	bot         *tgbotapi.BotAPI
	updatesCh   tgbotapi.UpdatesChannel
	rateLimiter ratelimit.Limiter
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
		token:       token,
		bot:         bot,
		updatesCh:   bot.GetUpdatesChan(u),
		rateLimiter: ratelimit.New(1),
	}, nil
}

func (c *client) GetUpdates() tgbotapi.UpdatesChannel {
	return c.updatesCh
}

func (c *client) SendResponse(ctx context.Context, response *domain.Response) {
	if response.Err != nil {
		slog.ErrorContext(ctx, "error occurred", "chatID", response.ChatID, logger.Err(response.Err))
		response.Text = fmt.Sprintf("❌ %s", response.Err.Error())
	}

	if response.Text != "" {
		if err := c.sendText(ctx, response.ChatID, response.Text); err != nil {
			c.handleError(ctx, response.ChatID, err)
		}
	}

	if response.Image != nil {
		if err := c.sendImage(ctx, response.ChatID, response.Image); err != nil {
			c.handleError(ctx, response.ChatID, err)
		}
	}

	if response.Keyboard != nil {
		if err := c.sendKeyboard(ctx, response.ChatID, response.Keyboard.Title, response.Keyboard.Buttons); err != nil {
			c.handleError(ctx, response.ChatID, err)
		}
	}
}

func (c *client) sendText(ctx context.Context, chatID int64, text string) error {
	htmlText := render.ToHTML(text)
	for len(htmlText) > 0 {
		if utf8.RuneCountInString(htmlText) <= maxTelegramMessageLength {
			return c.send(ctx, newHTMLMessage(chatID, htmlText))
		}

		cutIndex := c.findCutIndex(htmlText, maxTelegramMessageLength)
		if err := c.send(ctx, newHTMLMessage(chatID, htmlText[:cutIndex])); err != nil {
			return err
		}
		htmlText = htmlText[cutIndex:]
	}

	return nil
}

func newHTMLMessage(chatID int64, text string) tgbotapi.MessageConfig {
	m := tgbotapi.NewMessage(chatID, text)
	m.ParseMode = tgbotapi.ModeHTML
	m.DisableWebPagePreview = true
	return m
}

func (c *client) findCutIndex(text string, maxLength int) int {
	if i := strings.LastIndex(text[:maxLength], "<pre>"); i > -1 {
		return i
	}
	if i := strings.LastIndex(text[:maxLength], "\n"); i > -1 {
		return i
	}
	return maxLength
}

func (c *client) send(ctx context.Context, msg tgbotapi.Chattable) error {
	c.rateLimiter.Take()

	_, err := c.bot.Request(msg)
	if err != nil {
		slog.ErrorContext(ctx, "telegram send error", logger.Err(err))
	}
	return err
}

func (c *client) sendImage(ctx context.Context, chatID int64, image *domain.Image) error {
	callbackData := fmt.Sprintf("%s%d", domain.GenImageCallbackPrefix, image.PromptID)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Еще", callbackData),
		),
	)

	m := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Bytes: image.Data})
	m.ReplyMarkup = keyboard
	return c.send(ctx, m)
}

func (c *client) sendKeyboard(ctx context.Context, chatID int64, title string, buttons map[string]string) error {
	var row []tgbotapi.InlineKeyboardButton
	for label, callback := range buttons {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, callback))
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)

	m := tgbotapi.NewMessage(chatID, title)
	m.ReplyMarkup = keyboard
	return c.send(ctx, m)
}

func (c *client) AcknowledgeCallback(ctx context.Context, callbackQueryID string) {
	m := tgbotapi.NewCallback(callbackQueryID, "")
	_ = c.send(ctx, m)
}

func (c *client) handleError(ctx context.Context, chatID int64, err error) {
	slog.ErrorContext(ctx, "Error during sending message", logger.Err(err))

	errorMessage := fmt.Sprintf("❌ Не удалось доставить ответ:\n%v", err)
	m := tgbotapi.NewMessage(chatID, errorMessage)
	_ = c.send(ctx, m)
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

func (c *client) StartTyping(ctx context.Context, chatID int64) {
	m := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	c.send(ctx, m)
}
