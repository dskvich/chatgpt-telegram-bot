package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ChatRepository interface {
	Save(chat domain.Chat)
	GetByID(chatID int64) (domain.Chat, time.Time, bool)
	Clear(chatID int64)
}

type SettingsRepository interface {
	Save(ctx context.Context, settings domain.Settings) error
	GetByChatID(ctx context.Context, chatID int64) (*domain.Settings, error)
}

type chatService struct {
	chatRepo            ChatRepository
	settingsRepo        SettingsRepository
	supportedTextModels []string
	responseCh          chan<- domain.Response
}

func NewChatService(
	chatRepo ChatRepository,
	settingsRepo SettingsRepository,
	supportedTextModels []string,
	responseCh chan<- domain.Response,
) *chatService {
	return &chatService{
		chatRepo:            chatRepo,
		settingsRepo:        settingsRepo,
		supportedTextModels: supportedTextModels,
		responseCh:          responseCh,
	}
}

func (c *chatService) ClearChatHistory(ctx context.Context, chatID int64) {
	c.chatRepo.Clear(chatID)
	c.responseCh <- domain.Response{
		ChatID: chatID,
		Text:   "🧹 История очищена! Начните новый чат. 🚀",
	}
}

func (c *chatService) SendTextModels(ctx context.Context, chatID int64) {
	buttons := make(map[string]string, len(c.supportedTextModels))
	for _, model := range c.supportedTextModels {
		buttons[model] = domain.SetTextModelCallbackPrefix + model
	}

	c.responseCh <- domain.Response{
		ChatID: chatID,
		Keyboard: &domain.Keyboard{
			Title:   "⚙️ Выберите текстовую модель GPT:",
			Buttons: buttons,
		},
	}
}

func (c *chatService) SetTextModel(ctx context.Context, chatID int64, modelRaw string) {
	model, err := c.parseTextModel(modelRaw)
	if err != nil {
		c.responseCh <- domain.Response{ChatID: chatID, Err: err}
		return
	}

	settings, _ := c.settingsRepo.GetByChatID(ctx, chatID)
	if settings == nil {
		settings = &domain.Settings{ChatID: chatID}
	}
	settings.TextModel = model
	c.settingsRepo.Save(ctx, *settings)

	c.responseCh <- domain.Response{
		ChatID: chatID,
		Text:   "✅ Модель установлена: " + model,
	}

	c.ClearChatHistory(ctx, chatID)
}

func (c *chatService) parseTextModel(modelRaw string) (string, error) {
	if !strings.HasPrefix(modelRaw, domain.SetTextModelCallbackPrefix) {
		return "", fmt.Errorf("invalid format, expected prefix '%s'", domain.SetTextModelCallbackPrefix)
	}

	model := strings.TrimPrefix(modelRaw, domain.SetTextModelCallbackPrefix)

	for _, supportedModel := range c.supportedTextModels {
		if model == supportedModel {
			return model, nil
		}
	}

	return "", errors.New("unsupported model")
}

func (c *chatService) SendImageModels(ctx context.Context, chatID int64) {

}

func (c *chatService) SetImageModel(ctx context.Context, chatID int64, model string) {
	settings, _ := c.settingsRepo.GetByChatID(ctx, chatID)
	if settings == nil {
		settings = &domain.Settings{ChatID: chatID}
	}
	settings.ImageModel = model
	c.settingsRepo.Save(ctx, *settings)

	c.responseCh <- domain.Response{
		ChatID: chatID,
		Text:   "✅ Модель изображений установлена: " + model,
	}
}

func (c *chatService) SendTTLOptions(ctx context.Context, chatID int64) {
	const (
		ttlShort  = 15 * time.Minute
		ttlMedium = time.Hour
		ttlLong   = 8 * time.Hour
		ttlNone   = 0
	)

	chatTTLOptions := map[string]time.Duration{
		"ttl_15m":      ttlShort,
		"ttl_1h":       ttlMedium,
		"ttl_8h":       ttlLong,
		"ttl_disabled": ttlNone,
	}

	slog.DebugContext(ctx, "send ttl options", "chatTTLOptions", chatTTLOptions)

	// TODO: send keyboard
}

func (c *chatService) SetChatTTL(ctx context.Context, chatID int64, ttl string) {
	ttlDuration, err := time.ParseDuration(ttl)
	if err != nil {
		c.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("error parsing ttl duration: %w", err)}
		return
	}

	settings, _ := c.settingsRepo.GetByChatID(ctx, chatID)
	if settings == nil {
		settings = &domain.Settings{ChatID: chatID}
	}

	settings.TTL = ttlDuration
	c.settingsRepo.Save(ctx, *settings)

	c.responseCh <- domain.Response{
		ChatID: chatID,
		Text:   "✅ Время жизни чата (TTL) установлено: " + ttl,
	}
}

func (c *chatService) SendSystemPrompt(ctx context.Context, chatID int64) {
	// TODO: send prompt from settings
}

func (c *chatService) SetSystemPrompt(ctx context.Context, chatID int64, prompt string) {
	settings, _ := c.settingsRepo.GetByChatID(ctx, chatID)
	if settings == nil {
		settings = &domain.Settings{ChatID: chatID}
	}
	settings.SystemPrompt = prompt
	c.settingsRepo.Save(ctx, *settings)

	c.responseCh <- domain.Response{
		ChatID: chatID,
		Text:   "✅ Системный промпт установлен: " + prompt,
	}
}

func (c *chatService) SendGreeting(ctx context.Context, chatID int64) {
	greeting := `👋 Я твой ChatGPT Telegram-бот. Вот что умею:

❓ Отвечаю на вопросы. Напиши "/new" для очистки истории.
🎨 Рисую картинки. Начни запрос с "нарисуй".
🎙 Понимаю голосовые сообщения.
📷 Распознаю картинки.`

	c.responseCh <- domain.Response{
		ChatID: chatID,
		Text:   greeting,
	}
}

func (c *chatService) Save(ctx context.Context, chat domain.Chat) error {
	c.chatRepo.Save(chat)

	return nil
}

func (c *chatService) GetChatByID(ctx context.Context, chatID int64) (*domain.Chat, error) {
	settings, err := c.settingsRepo.GetByChatID(ctx, chatID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			slog.WarnContext(ctx, "Settings not found, using defaults", "chatID", chatID)
			settings = &domain.Settings{}
		} else {
			return nil, fmt.Errorf("fetching settings: %w", err)
		}
	}

	if settings.TextModel == "" {
		settings.TextModel = domain.Gpt4oMiniModel
	}
	if settings.TTL == 0 {
		settings.TTL = time.Minute * 15
	}

	chat, lastUpdate, ok := c.chatRepo.GetByID(chatID)
	if ok && !c.isExpired(lastUpdate, settings.TTL) {
		return &chat, nil
	}

	slog.DebugContext(ctx, "Creating a new chat with parameters",
		"textModel", settings.TextModel,
		"ttl", settings.TTL,
		"systemPrompt", settings.SystemPrompt,
	)

	var messages []domain.ChatMessage
	if settings.SystemPrompt != "" {
		messages = append(messages, domain.ChatMessage{
			Role:    "developer",
			Content: settings.SystemPrompt,
		})
	}

	c.responseCh <- domain.Response{
		ChatID: chatID,
		Text: fmt.Sprintf(`<i>🛠️ Создан новый чат!
		Текстовая модель GPT: %s
		Период хранения данных: %s
		Системная инструкция: %s
		</i>`, settings.TextModel, settings.TTL, settings.SystemPrompt),
	}

	return &domain.Chat{
		ID:        chatID,
		ModelName: settings.TextModel,
		Messages:  messages,
	}, nil
}

func (c *chatService) isExpired(lastUpdate time.Time, ttl time.Duration) bool {
	if ttl <= 0 {
		return false
	}
	return time.Since(lastUpdate) > ttl
}
