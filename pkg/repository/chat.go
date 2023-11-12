package repository

import (
	"sync"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type chatRepository struct {
	mu       sync.RWMutex
	sessions map[int64]domain.ChatSession
}

func NewChatRepository() *chatRepository {
	return &chatRepository{
		sessions: make(map[int64]domain.ChatSession),
	}
}

func (c *chatRepository) SaveSession(chatID int64, session domain.ChatSession) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sessions[chatID] = session
}

func (c *chatRepository) GetSession(chatID int64) (domain.ChatSession, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	session, ok := c.sessions[chatID]
	return session, ok
}

func (c *chatRepository) RemoveSession(chatID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.sessions, chatID)
}
