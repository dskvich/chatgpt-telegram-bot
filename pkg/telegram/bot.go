package telegram

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/logger"
)

type bot struct {
	token   string
	api     *tgbotapi.BotAPI
	updates tgbotapi.UpdatesChannel
}

func NewBot(token string) (*bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating bot api: %v", err)
	}

	slog.Info("authorized on telegram", "bot", api.Self)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	return &bot{
		token:   token,
		api:     api,
		updates: api.GetUpdatesChan(u),
	}, nil
}

func (b *bot) GetUpdates() tgbotapi.UpdatesChannel {
	return b.updates
}

func (b *bot) Send(message domain.Message) error {
	if _, err := b.api.Send(message.ToChatMessage()); err != nil {
		return fmt.Errorf("sending message: %v", err)
	}
	return nil
}

func (b *bot) DownloadFile(fileID string) (string, error) {
	file, err := b.api.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("getting file: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, file.Link(b.token), nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %v", err)
	}

	resp, err := b.api.Client.Do(req)
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
