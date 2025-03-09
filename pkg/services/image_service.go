package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type OpenAIImageGenerator interface {
	GenerateImage(ctx context.Context, prompt string) ([]byte, error)
}

type PromptsRepository interface {
	Save(ctx context.Context, prompt string) (int64, error)
	GetByID(ctx context.Context, id int64) (string, error)
}

type imageService struct {
	openAIImageGenerator OpenAIImageGenerator
	promptsRepo          PromptsRepository
	responseCh           chan<- domain.Response
}

func NewImageService(
	openAIImageGenerator OpenAIImageGenerator,
	promptsRepo PromptsRepository,
	responseCh chan<- domain.Response,
) *imageService {
	return &imageService{
		openAIImageGenerator: openAIImageGenerator,
		promptsRepo:          promptsRepo,
		responseCh:           responseCh,
	}
}

func (s *imageService) GenerateImage(ctx context.Context, chatID int64, prompt string) {
	slog.InfoContext(ctx, "Starting image generation", "prompt", prompt)

	promptID, err := s.promptsRepo.Save(ctx, prompt)
	if err != nil {
		s.responseCh <- domain.Response{ChatID: chatID, Err: err}
		return
	}

	slog.InfoContext(ctx, "Prompt saved", "promptID", promptID)

	imageData, err := s.openAIImageGenerator.GenerateImage(ctx, prompt)
	if err != nil {
		s.responseCh <- domain.Response{ChatID: chatID, Err: err}
		return
	}

	slog.InfoContext(ctx, "Image generated", "size", len(imageData))

	s.responseCh <- domain.Response{
		ChatID: chatID,
		Image: &domain.Image{
			PromptID: promptID,
			Data:     imageData,
		},
	}
}

func (s *imageService) GenerateImageByPromptID(ctx context.Context, chatID int64, imageIDRaw string) {
	slog.InfoContext(ctx, "Starting generation image by imageID", "imageIDRaw", imageIDRaw)

	imageID, err := s.parseImageID(imageIDRaw)
	if err != nil {
		s.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("parsing image PromptID: %w", err)}
		return
	}

	slog.InfoContext(ctx, "ImageID parsed", "imageID", imageID)

	prompt, err := s.promptsRepo.GetByID(ctx, imageID)
	if err != nil {
		s.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("getting prompt: %w", err)}
		return
	}

	slog.InfoContext(ctx, "Prompt fetched", "prompt", prompt)

	imageData, err := s.openAIImageGenerator.GenerateImage(ctx, prompt)
	if err != nil {
		s.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("generating image: %w", err)}
		return
	}

	slog.InfoContext(ctx, "Image generated", "size", len(imageData))

	s.responseCh <- domain.Response{
		ChatID: chatID,
		Image: &domain.Image{
			PromptID: imageID,
			Data:     imageData,
		},
	}
}

func (s *imageService) parseImageID(imageIDRaw string) (int64, error) {
	idStr := strings.TrimPrefix(imageIDRaw, domain.GenImageCallbackPrefix)

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid imageID: %s", imageIDRaw)
	}

	return id, nil
}
