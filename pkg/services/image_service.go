package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type OpenAIImageGenerator interface {
	GenerateImage(ctx context.Context, prompt, style string) ([]byte, error)
}

type ImagePromptsRepository interface {
	Save(ctx context.Context, i *domain.ImagePrompt) (int64, error)
	GetPrompt(ctx context.Context, imageID int64) (string, error)
}

type imageService struct {
	openAIImageGenerator OpenAIImageGenerator
	imagePromptsRepo     ImagePromptsRepository
	settingsRepo         SettingsRepository
}

func NewImageService(
	openAIImageGenerator OpenAIImageGenerator,
	imagePromptsRepo ImagePromptsRepository,
	settingsRepo SettingsRepository,
) *imageService {
	return &imageService{
		openAIImageGenerator: openAIImageGenerator,
		imagePromptsRepo:     imagePromptsRepo,
		settingsRepo:         settingsRepo,
	}
}

func (s *imageService) SetImageStyle(ctx context.Context, chatID int64, style string) error {
	if err := s.settingsRepo.Save(ctx, chatID, domain.ImageStyleKey, style); err != nil {
		return fmt.Errorf("saving image style: %w", err)
	}

	return nil
}

func (s *imageService) GenerateImage(ctx context.Context, chatID int64, prompt, user string) (*domain.Image, error) {
	slog.InfoContext(ctx, "Starting image generation", "prompt", prompt)

	imagePrompt := &domain.ImagePrompt{
		ChatID:   chatID,
		Prompt:   prompt,
		FromUser: user,
	}

	imageID, err := s.imagePromptsRepo.Save(ctx, imagePrompt)
	if err != nil {
		return nil, fmt.Errorf("saving image prompt: %w", err)
	}

	slog.InfoContext(ctx, "Image prompt saved successfully", "imageID", imageID)

	settings, err := s.settingsRepo.GetAll(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("getting system setting: %w", err)
	}

	style, found := settings[domain.ImageStyleKey]
	if !found {
		style = domain.ImageStyleDefault
	}

	slog.InfoContext(ctx, "Sending request to generate image", "style", style)

	imageData, err := s.openAIImageGenerator.GenerateImage(ctx, prompt, style)
	if err != nil {
		return nil, fmt.Errorf("generating image: %w", err)
	}

	slog.InfoContext(ctx, "Image generated successfully", "imageBytes", len(imageData))

	return &domain.Image{
		ID:   imageID,
		Data: imageData,
	}, nil
}

func (s *imageService) GenerateImageByID(ctx context.Context, chatID, imageID int64) (*domain.Image, error) {
	slog.InfoContext(ctx, "Starting generation image by imageID", "imageID", imageID)

	prompt, err := s.imagePromptsRepo.GetPrompt(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("getting prompt: %w", err)
	}

	slog.InfoContext(ctx, "Prompt fetched successfully", "prompt", prompt)

	settings, err := s.settingsRepo.GetAll(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("getting system setting: %w", err)
	}

	style, found := settings[domain.ImageStyleKey]
	if !found {
		style = domain.ImageStyleDefault
	}

	slog.InfoContext(ctx, "Sending request to generate image", "style", style)

	imageData, err := s.openAIImageGenerator.GenerateImage(ctx, prompt, style)
	if err != nil {
		return nil, fmt.Errorf("generating image: %w", err)
	}

	slog.InfoContext(ctx, "Image generated successfully", "imageBytes", len(imageData))

	return &domain.Image{
		ID:   imageID,
		Data: imageData,
	}, nil
}
