package domain

import "time"

type Settings struct {
	ChatID       int64
	TextModel    string
	SystemPrompt string
	ImageModel   string
	TTL          time.Duration
}
