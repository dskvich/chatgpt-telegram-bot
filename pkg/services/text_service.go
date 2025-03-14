package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type OpenAIClient interface {
	CreateChatCompletion(ctx context.Context, chat *domain.Chat) (domain.ChatMessage, error)
	TranscribeAudio(ctx context.Context, audioFilePath string) (string, error)
}

type ImageFileDownloader interface {
	DownloadFile(ctx context.Context, fileID string) ([]byte, error)
}

type textService struct {
	openAIClient OpenAIClient
	imageService *imageService
	chatService  *chatService
	downloader   ImageFileDownloader
	responseCh   chan<- domain.Response
}

func NewTextService(
	openAIClient OpenAIClient,
	imageService *imageService,
	chatService *chatService,
	downloader ImageFileDownloader,
	responseCh chan<- domain.Response,
) *textService {
	return &textService{
		openAIClient: openAIClient,
		imageService: imageService,
		chatService:  chatService,
		downloader:   downloader,
		responseCh:   responseCh,
	}
}

func (t *textService) GenerateFromText(ctx context.Context, chatID int64, prompt string) {
	t.generateTextResponse(ctx, chatID, nil, prompt)
}

func (t *textService) GenerateFromImage(ctx context.Context, chatID int64, imageFileID, prompt string) {
	imageData, err := t.downloader.DownloadFile(ctx, imageFileID)
	if err != nil {
		t.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("downloading image file: %w", err)}
		return
	}

	t.generateTextResponse(ctx, chatID, imageData, prompt)
}

func (t *textService) generateTextResponse(ctx context.Context, chatID int64, imageData []byte, prompt string) {
	slog.InfoContext(ctx, "Generating text response", "prompt", prompt, "imageDataSizeBytes", len(imageData))

	chat, err := t.chatService.GetChatByID(ctx, chatID)
	if err != nil {
		t.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("fetching chat by id: %w", err)}
		return
	}

	// Add user message
	chat.Messages = append(chat.Messages, domain.ChatMessage{
		Role:    "user",
		Content: t.prepareContent(prompt, imageData),
	})

	slog.InfoContext(ctx, "Calling OpenAI for chat completion", "model", chat.ModelName, "messagesCount", len(chat.Messages))

	chatResponse, err := t.openAIClient.CreateChatCompletion(ctx, chat)
	if err != nil {
		t.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("creating chat completion: %w", err)}
		return
	}

	slog.DebugContext(ctx, "Chat completion received", "content", chatResponse.Content)

	// Add assistant message
	chat.Messages = append(chat.Messages, chatResponse)

	if chatResponse.Content != nil {
		t.chatService.Save(ctx, *chat)

		t.responseCh <- domain.Response{ChatID: chatID, Text: fmt.Sprint(chatResponse.Content)}
		return
	}

	t.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("unexpected chat completion response: %+v", chatResponse)}
	return
}

func (t *textService) prepareContent(text string, image []byte) any {
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
