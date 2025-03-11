package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/samber/lo"

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
	supportedTTLOptions []time.Duration
	responseCh          chan<- domain.Response
}

func NewChatService(
	chatRepo ChatRepository,
	settingsRepo SettingsRepository,
	supportedTextModels []string,
	supportedTTLOptions []time.Duration,
	responseCh chan<- domain.Response,
) *chatService {
	return &chatService{
		chatRepo:            chatRepo,
		settingsRepo:        settingsRepo,
		supportedTextModels: supportedTextModels,
		supportedTTLOptions: supportedTTLOptions,
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
	c.responseCh <- domain.Response{
		ChatID: chatID,
		Keyboard: &domain.Keyboard{
			Title:          "⚙️ Выберите текстовую модель GPT:",
			ButtonLabels:   c.supportedTextModels,
			CallbackPrefix: domain.SetTextModelCallbackPrefix,
			ButtonsPerRow:  2,
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
	settings, _ = lo.Coalesce(settings, &domain.Settings{ChatID: chatID})
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

	if lo.Contains(c.supportedTextModels, model) {
		return model, nil
	}

	return "", errors.New("unsupported model")
}

func (c *chatService) SendImageModels(ctx context.Context, chatID int64) {
	c.responseCh <- domain.Response{
		ChatID: chatID,
		Text:   "🚧 В разработке 🚧",
	}
}

func (c *chatService) SetImageModel(ctx context.Context, chatID int64, model string) {
	settings, _ := c.settingsRepo.GetByChatID(ctx, chatID)
	settings, _ = lo.Coalesce(settings, &domain.Settings{ChatID: chatID})
	settings.ImageModel = model
	c.settingsRepo.Save(ctx, *settings)

	c.responseCh <- domain.Response{
		ChatID: chatID,
		Text:   "✅ Модель изображений установлена: " + model,
	}
}

func (c *chatService) SendTTLOptions(ctx context.Context, chatID int64) {
	ttlOptions := lo.Map(c.supportedTTLOptions, func(d time.Duration, _ int) string {
		return c.shortDuration(d)
	})

	c.responseCh <- domain.Response{
		ChatID: chatID,
		Keyboard: &domain.Keyboard{
			Title:          "⚙️ Выберите период хранения данных чата:",
			ButtonLabels:   ttlOptions,
			CallbackPrefix: domain.SetTTLCallbackPrefix,
			ButtonsPerRow:  10,
		},
	}
}

func (c *chatService) shortDuration(d time.Duration) string {
	s := d.String()
	s = lo.Ternary(strings.HasSuffix(s, "m0s"), s[:len(s)-2], s)
	s = lo.Ternary(strings.HasSuffix(s, "h0m"), s[:len(s)-2], s)
	return s
}

func (c *chatService) SetChatTTL(ctx context.Context, chatID int64, ttlRaw string) {
	ttl, err := c.parseTTL(ttlRaw)
	if err != nil {
		c.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("error parsing ttl duration: %w", err)}
		return
	}

	settings, _ := c.settingsRepo.GetByChatID(ctx, chatID)
	settings, _ = lo.Coalesce(settings, &domain.Settings{ChatID: chatID})
	settings.TTL = ttl
	c.settingsRepo.Save(ctx, *settings)

	c.responseCh <- domain.Response{
		ChatID: chatID,
		Text:   "✅ Время жизни чата (TTL) установлено: " + c.shortDuration(ttl),
	}
}

func (c *chatService) parseTTL(ttlRaw string) (time.Duration, error) {
	if !strings.HasPrefix(ttlRaw, domain.SetTTLCallbackPrefix) {
		return 0, fmt.Errorf("invalid format, expected prefix '%s'", domain.SetTTLCallbackPrefix)
	}

	ttlStr := strings.TrimPrefix(ttlRaw, domain.SetTTLCallbackPrefix)

	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		return 0, err
	}

	if lo.Contains(c.supportedTTLOptions, ttl) {
		return ttl, nil
	}

	return 0, errors.New("unsupported ttl option")
}

func (c *chatService) SendSystemPrompt(ctx context.Context, chatID int64) {
	c.responseCh <- domain.Response{
		ChatID: chatID,
		Text:   "🚧 В разработке 🚧",
	}
}

func (c *chatService) SetSystemPrompt(ctx context.Context, chatID int64, prompt string) {
	settings, _ := c.settingsRepo.GetByChatID(ctx, chatID)
	settings, _ = lo.Coalesce(settings, &domain.Settings{ChatID: chatID})
	settings.SystemPrompt = prompt
	c.settingsRepo.Save(ctx, *settings)

	c.responseCh <- domain.Response{
		ChatID: chatID,
		Text:   "✅ Системный промпт установлен: " + prompt,
	}
}

func (c *chatService) SendGreeting(ctx context.Context, chatID int64) {
	greeting := `👋 Привет! Я твой ChatGPT Telegram-бот. Вот что я умею:

🆕 **/new** — Начать новый чат
⏳ **/ttl** — Установить время жизни чата
📝 **/text_models** — Выбрать модель для текста
🖼️ **/image_models** — Выбрать модель для картинок
⚙️ **/system_prompt** — Настроить системную инструкцию

🖊️ Просто задай мне вопрос — я помогу!
🎨 Напиши "нарисуй ..." и я создам картинку.
🎙 Отправь голосовое сообщение — я пойму.
📷 Отправь картинку — я опишу её или отвечу на твои вопросы о ней.

Начнем? 🚀`

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
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("fetching settings: %w", err)
	}

	settings, _ = lo.Coalesce(settings, &domain.Settings{})
	settings.TextModel, _ = lo.Coalesce(settings.TextModel, domain.Gpt4oMiniModel)
	settings.TTL, _ = lo.Coalesce(settings.TTL, 15*time.Minute)

	chat, lastUpdate, ok := c.chatRepo.GetByID(chatID)
	if ok && !c.isExpired(lastUpdate, settings.TTL) {
		return &chat, nil
	}

	slog.DebugContext(ctx, "Creating a new chat with parameters",
		"textModel", settings.TextModel,
		"ttl", settings.TTL,
		"systemPrompt", settings.SystemPrompt,
	)

	messages := lo.If(settings.SystemPrompt != "",
		[]domain.ChatMessage{{Role: "developer", Content: settings.SystemPrompt}}).
		Else(nil)

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
