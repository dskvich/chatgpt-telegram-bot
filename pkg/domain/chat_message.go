package domain

type ChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // Content can be a string or a slice
}
