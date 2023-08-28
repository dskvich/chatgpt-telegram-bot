package repository

import (
	"sync"

	"github.com/sashabaranov/go-openai"
)

type chatRepository struct {
	mu       sync.RWMutex
	messages map[int64][]openai.ChatCompletionMessage
}

func NewChatRepository() *chatRepository {
	return &chatRepository{
		messages: make(map[int64][]openai.ChatCompletionMessage),
	}
}

func (c *chatRepository) AddMessage(chatID int64, msg openai.ChatCompletionMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.messages[chatID] = append(c.messages[chatID], msg)
}

func (c *chatRepository) GetMessages(chatID int64) []openai.ChatCompletionMessage {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.messages[chatID]) == 0 {
		return nil
	}

	res := make([]openai.ChatCompletionMessage, 0, len(c.messages[chatID]))
	for _, msg := range c.messages[chatID] {
		res = append(res, msg)
	}

	return res
}

func (c *chatRepository) RemoveMessages(chatID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.messages[chatID] = nil
}
