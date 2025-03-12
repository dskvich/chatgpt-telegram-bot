package telegram

import (
	"context"
	"log/slog"
	"strings"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/keyword"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ImageService interface {
	GenerateImageByPromptID(ctx context.Context, chatID int64, promptID string)
	GenerateImage(ctx context.Context, chatID int64, prompt string)
}

type TextService interface {
	GenerateFromText(ctx context.Context, chatID int64, prompt string)
	GenerateFromImage(ctx context.Context, chatID int64, imageFileID, prompt string)
}

type VoiceService interface {
	GenerateFromVoice(ctx context.Context, chatID int64, voiceFileID string)
}

type ChatService interface {
	ClearChatHistory(ctx context.Context, chatID int64)
	SetTextModel(ctx context.Context, chatID int64, model string)
	SetImageModel(ctx context.Context, chatID int64, model string)
	SetChatTTL(ctx context.Context, chatID int64, ttl string)
	SetSystemPrompt(ctx context.Context, chatID int64, prompt string)
	SendGreeting(ctx context.Context, chatID int64)
	SendTextModels(ctx context.Context, chatID int64)
	SendImageModels(ctx context.Context, chatID int64)
	SendTTLOptions(ctx context.Context, chatID int64)
	SendSystemPrompt(ctx context.Context, chatID int64)
	HasState(chatID int64) bool
	HandleState(ctx context.Context, chatID int64, text string)
	RequestSystemPromptUpdate(ctx context.Context, chatID int64)
}

type handler struct {
	imageService ImageService
	textService  TextService
	voiceService VoiceService
	chatService  ChatService
}

func NewHandler(
	imageService ImageService,
	textService TextService,
	voiceService VoiceService,
	chatService ChatService,
) *handler {
	return &handler{
		imageService: imageService,
		textService:  textService,
		voiceService: voiceService,
		chatService:  chatService,
	}
}

func (h *handler) HandleUpdate(ctx context.Context, update *tgbotapi.Update) {
	switch {
	case update.CallbackQuery != nil:
		h.handleCallback(ctx, update.CallbackQuery)

	case update.Message != nil:
		h.handleMessage(ctx, update.Message)
	}
}

func (h *handler) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	switch {
	case strings.HasPrefix(data, domain.GenImageCallbackPrefix):
		h.imageService.GenerateImageByPromptID(ctx, chatID, data)
	case strings.HasPrefix(data, domain.SetTextModelCallbackPrefix):
		h.chatService.SetTextModel(ctx, chatID, data)
	case strings.HasPrefix(data, domain.SetImageModelCallbackPrefix):
		h.chatService.SetImageModel(ctx, chatID, data)
	case strings.HasPrefix(data, domain.SetTTLCallbackPrefix):
		h.chatService.SetChatTTL(ctx, chatID, data)
	case strings.HasPrefix(data, domain.SetSystemPromptCallbackPrefix):
		h.chatService.RequestSystemPromptUpdate(ctx, chatID)
	default:
		slog.Warn("Unhandled callback", "data", data)
	}
}

func (h *handler) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	// Check chat state, like editing system prompt when
	// the next text message should be interpreted as the new system prompt.
	if h.chatService.HasState(msg.Chat.ID) {
		h.chatService.HandleState(ctx, msg.Chat.ID, msg.Text)
		return
	}

	switch {
	case msg.Photo != nil:
		h.textService.GenerateFromImage(ctx, msg.Chat.ID, msg.Photo[len(msg.Photo)-1].FileID, msg.Caption)

	case msg.Voice != nil:
		h.voiceService.GenerateFromVoice(ctx, msg.Chat.ID, msg.Voice.FileID)

	case isCommand(msg.Text):
		h.handleCommand(ctx, msg.Chat.ID, msg.Text)

	case keyword.IsImageRequest(msg.Text):
		h.imageService.GenerateImage(ctx, msg.Chat.ID, msg.Text)

	default:
		h.textService.GenerateFromText(ctx, msg.Chat.ID, msg.Text)
	}
}

func isCommand(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "/")
}

func (h *handler) handleCommand(ctx context.Context, chatID int64, text string) {
	cmd := strings.ToLower(strings.TrimSpace(text))
	cmd = strings.Split(cmd, "@")[0]

	switch {
	case cmd == "/start":
		h.chatService.SendGreeting(ctx, chatID)

	case cmd == "/new":
		h.chatService.ClearChatHistory(ctx, chatID)

	case strings.HasPrefix(cmd, "/image_models"):
		h.chatService.SendImageModels(ctx, chatID)

	case strings.HasPrefix(cmd, "/text_models"):
		h.chatService.SendTextModels(ctx, chatID)

	case strings.HasPrefix(cmd, "/system_prompt"):
		h.chatService.SendSystemPrompt(ctx, chatID)

	case strings.HasPrefix(cmd, "/ttl"):
		h.chatService.SendTTLOptions(ctx, chatID)

	default:
		slog.Warn("Unhandled command", "cmd", cmd)
	}
}
