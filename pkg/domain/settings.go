package domain

import "time"

type Settings struct {
	ChatID       int64
	TopicID      int
	TextModel    string
	SystemPrompt string
	ImageModel   string
	TTL          time.Duration
}
