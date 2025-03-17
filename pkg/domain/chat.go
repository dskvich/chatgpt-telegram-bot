package domain

type Chat struct {
	ID        int64
	TopicID   int
	ModelName string
	Messages  []ChatMessage
}
