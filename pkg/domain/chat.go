package domain

import "time"

type Chat struct {
	ID           int64
	TopicID      int
	Model        string
	TTL          time.Duration
	SystemPrompt string
	Messages     []Message
}

type Message struct {
	Role         string
	ContentParts []ContentPart
}

const MessageRoleUser = "user"

type ContentPart struct {
	Type ContentPartType
	Data string
}

type ContentPartType string

const (
	ContentPartTypeText  ContentPartType = "text"
	ContentPartTypeImage ContentPartType = "image"
)
