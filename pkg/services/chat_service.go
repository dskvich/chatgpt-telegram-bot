package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

const (
	voiceTempDirPerm = 0o755  // Permissions for the temporary directory
	voiceFilePerm    = 0o600  // Permissions for temporary files
	voiceTempDir     = "temp" // Directory for temporary files
)

type OpenAIClient interface {
	CreateChatCompletion(ctx context.Context, chat *domain.Chat) (domain.ChatMessage, error)
	TranscribeAudio(ctx context.Context, audioFilePath string) (string, error)
}

type ChatRepository interface {
	Save(chat domain.Chat)
	GetByID(chatID int64) (domain.Chat, bool)
	ClearChat(chatID int64)
	SetTTL(chatID int64, ttl time.Duration)
}

type SettingsRepository interface {
	GetAll(ctx context.Context, chatID int64) (map[string]string, error)
	Save(ctx context.Context, chatID int64, key, value string) error
}

type ChatStyleRepository interface {
	GetActiveStyle(ctx context.Context, chatID int64) (*domain.ChatStyle, error)
	GetAllStyles(ctx context.Context, chatID int64) ([]domain.ChatStyle, error)
}

type ToolService interface {
	Tools() []domain.Tool
	InvokeFunction(ctx context.Context, chatID int64, name, args string) (string, error)
}

type IntentDetector interface {
	DetectIntent(prompt string) domain.Intent
}

type AudioConverter interface {
	ConvertToMP3(inputPath string) (string, error)
}

type ImageService interface {
	GenerateImage(ctx context.Context, chatID int64, prompt, user string) (*domain.Image, error)
}

type ChatService struct {
	openAIClient   OpenAIClient
	chatRepo       ChatRepository
	settingsRepo   SettingsRepository
	chatStyleRepo  ChatStyleRepository
	toolService    ToolService
	intentDetector IntentDetector
	ttlOptions     map[string]time.Duration
	audioConverter AudioConverter
	imageService   ImageService
}

func NewChatService(
	openAIClient OpenAIClient,
	chatRepo ChatRepository,
	settingsRepo SettingsRepository,
	chatStyleRepo ChatStyleRepository,
	toolService ToolService,
	intentDetector IntentDetector,
	ttlOptions map[string]time.Duration,
	audioConverter AudioConverter,
	imageService ImageService,
) *ChatService {
	return &ChatService{
		openAIClient:   openAIClient,
		chatRepo:       chatRepo,
		settingsRepo:   settingsRepo,
		chatStyleRepo:  chatStyleRepo,
		toolService:    toolService,
		intentDetector: intentDetector,
		ttlOptions:     ttlOptions,
		audioConverter: audioConverter,
		imageService:   imageService,
	}
}

func (c *ChatService) SetChatTTL(ctx context.Context, chatID int64, ttl time.Duration) error {
	c.chatRepo.SetTTL(chatID, ttl)
	return nil
}

func (c *ChatService) GetChatStyles(ctx context.Context, chatID int64) ([]domain.ChatStyle, error) {
	return c.chatStyleRepo.GetAllStyles(ctx, chatID)
}

func (c *ChatService) GetChatSettings(ctx context.Context, chatID int64) (map[string]string, error) {
	return c.settingsRepo.GetAll(ctx, chatID)
}

func (c *ChatService) ClearChatHistory(ctx context.Context, chatID int64) error {
	c.chatRepo.ClearChat(chatID)
	return nil
}

func (c *ChatService) GenerateResponseFromVoice(ctx context.Context, chatID int64, voiceData []byte) (*domain.Response, error) {
	voiceFilePath := filepath.Join(voiceTempDir, fmt.Sprintf("voice_%d.ogg", time.Now().UnixNano()))

	if err := os.MkdirAll(voiceTempDir, voiceTempDirPerm); err != nil {
		return nil, fmt.Errorf("creating temp directory: %w", err)
	}

	if err := os.WriteFile(voiceFilePath, voiceData, voiceFilePerm); err != nil {
		return nil, fmt.Errorf("saving voice file: %w", err)
	}

	mp3Path, err := c.audioConverter.ConvertToMP3(voiceFilePath)
	if err != nil {
		return nil, fmt.Errorf("converting voice file to MP3: %w", err)
	}

	transcription, err := c.openAIClient.TranscribeAudio(ctx, mp3Path)
	if err != nil {
		return nil, fmt.Errorf("transcribing audio file: %w", err)
	}

	return c.GenerateResponse(ctx, chatID, nil, transcription)
}

func (c *ChatService) GenerateResponse(ctx context.Context, chatID int64, imageData []byte, prompt string) (*domain.Response, error) {
	intent := c.intentDetector.DetectIntent(prompt)

	switch intent {
	case domain.IntentGenerateImage:
		image, err := c.imageService.GenerateImage(ctx, chatID, prompt, "")
		if err != nil {
			return nil, fmt.Errorf("generating image: %w", err)
		}

		return &domain.Response{Image: image}, nil
	case domain.IntentGenerateText:
		response, err := c.generateTextResponse(ctx, chatID, imageData, prompt)
		if err != nil {
			return nil, fmt.Errorf("generating response: %w", err)
		}
		return response, nil
	default:
		return nil, errors.New("unable to determine intent of the message")
	}
}

func (c *ChatService) generateTextResponse(ctx context.Context, chatID int64, imageData []byte, prompt string) (*domain.Response, error) {
	slog.InfoContext(ctx, "Generating text response", "prompt", prompt, "imageDataLen", len(imageData))

	chat, err := c.getOrCreateChat(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("getting or creating chat: %w", err)
	}

	// Add user message
	chat.Messages = append(chat.Messages, domain.ChatMessage{
		Role:    "user",
		Content: c.prepareContent(prompt, imageData),
	})

	slog.InfoContext(ctx, "Calling OpenAI for chat completion", "model", chat.ModelName, "messagesCount", len(chat.Messages))

	chatResponse, err := c.openAIClient.CreateChatCompletion(ctx, chat)
	if err != nil {
		return nil, fmt.Errorf("creating chat completion: %w", err)
	}

	slog.DebugContext(ctx, "Chat completion received", "content", chatResponse.Content, "toolCallsCount", len(chatResponse.ToolCalls))

	// Add assistant message
	chat.Messages = append(chat.Messages, chatResponse)

	if chatResponse.Content != nil {
		c.chatRepo.Save(*chat)
		return &domain.Response{Text: fmt.Sprint(chatResponse.Content)}, nil
	}

	// Process tool invocations, if any
	for _, toolCall := range chatResponse.ToolCalls {
		toolResponse, err := c.toolService.InvokeFunction(ctx, chat.ID, toolCall.Function.Name, toolCall.Function.Arguments)
		if err != nil {
			return nil, fmt.Errorf("invoking tool: %w", err)
		}

		// Add tool message
		chat.Messages = append(chat.Messages, domain.ChatMessage{
			ToolCallID: toolCall.ID,
			Role:       "tool",
			Name:       toolCall.Function.Name,
			Content:    toolResponse,
		})

		slog.InfoContext(ctx, "Calling OpenAI for post-tool chat completion", "messagesCount", len(chat.Messages))

		afterToolResponse, err := c.openAIClient.CreateChatCompletion(ctx, chat)
		if err != nil {
			return nil, fmt.Errorf("creating chat completion after invoking tool: %w", err)
		}

		slog.DebugContext(ctx, "Post-tool chat completion received", "responseContent", afterToolResponse.Content)

		// Add assistant message
		chat.Messages = append(chat.Messages, afterToolResponse)

		c.chatRepo.Save(*chat)
		return &domain.Response{Text: fmt.Sprint(afterToolResponse.Content)}, nil
	}

	return nil, fmt.Errorf("unexpected chat completion response: %+v", chatResponse)
}

func (c *ChatService) prepareContent(text string, image []byte) any {
	if image != nil {
		imageContent := domain.Content{
			Type: "image_url",
			ImageURL: &domain.ImageURL{
				URL: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(image),
			},
		}
		if text != "" {
			return []domain.Content{{Type: "text", Text: text}, imageContent}
		}
		return []domain.Content{imageContent}
	}
	return text
}

func (c *ChatService) getOrCreateChat(ctx context.Context, chatID int64) (*domain.Chat, error) {
	if chat, ok := c.chatRepo.GetByID(chatID); ok {
		return &chat, nil
	}

	slog.DebugContext(ctx, "Chat not found, creating a new one")

	// create a new chat
	settings, err := c.settingsRepo.GetAll(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch settings: %w", err)
	}

	model := settings[domain.ModelKey]
	if model == "" {
		model = domain.DefaultModel
	}

	style, err := c.chatStyleRepo.GetActiveStyle(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch active style: %w", err)
	}

	var messages []domain.ChatMessage
	if style != nil && style.Description != "" {
		messages = append(messages, domain.ChatMessage{
			Role:    "developer",
			Content: style.Description,
		})
		slog.DebugContext(ctx, "Added developer instructions", "description", style.Description)
	}

	tools := c.toolService.Tools()
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Function.Name
	}
	slog.DebugContext(ctx, "Tools available", "toolNames", toolNames)

	return &domain.Chat{
		ID:        chatID,
		ModelName: model,
		Messages:  messages,
		Tools:     tools,
	}, nil
}
