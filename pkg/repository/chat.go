package repository

import (
	"sync"
	"time"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type sessionEntry struct {
	session    domain.ChatSession
	lastUpdate time.Time
}

type chatRepository struct {
	mu       sync.RWMutex
	sessions map[int64]sessionEntry
	ttl      time.Duration
}

func NewChatRepository() *chatRepository {
	return &chatRepository{
		sessions: make(map[int64]sessionEntry),
		ttl:      15 * time.Minute,
	}
}

func (c *chatRepository) SaveSession(chatID int64, session domain.ChatSession) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clean up expired session before saving a new one
	if entry, ok := c.sessions[chatID]; ok {
		if time.Since(entry.lastUpdate) > c.ttl {
			delete(c.sessions, chatID)
		}
	}

	c.sessions[chatID] = sessionEntry{
		session:    session,
		lastUpdate: time.Now(),
	}
}

func (c *chatRepository) GetSession(chatID int64) (domain.ChatSession, bool) {
	c.mu.RLock()
	entry, ok := c.sessions[chatID]
	c.mu.RUnlock()

	if !ok || time.Since(entry.lastUpdate) > c.ttl {
		return domain.ChatSession{}, false
	}

	return entry.session, true
}

func (c *chatRepository) RemoveSession(chatID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.sessions, chatID)
}
