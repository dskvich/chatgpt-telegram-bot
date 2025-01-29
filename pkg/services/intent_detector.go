package services

import (
	"strings"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type intentDetector struct {
	imageKeywords []string
}

func NewIntentDetector(imageKeywords []string) *intentDetector {
	return &intentDetector{
		imageKeywords: imageKeywords,
	}
}

func (i *intentDetector) DetectIntent(prompt string) domain.Intent {
	// Check if the prompt contains image-generation-related keywords
	lowerText := strings.ToLower(prompt)
	for _, keyword := range i.imageKeywords {
		if strings.Contains(lowerText, keyword) {
			return domain.IntentGenerateImage
		}
	}

	// Default to text generation
	return domain.IntentGenerateText
}
