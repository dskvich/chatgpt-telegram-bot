package domain

type Chat struct {
	ID        int64
	ModelName string
	Messages  []ChatMessage
}
