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
	mu         sync.RWMutex
	sessions   map[int64]sessionEntry
	chatTTL    map[int64]time.Duration
	defaultTTL time.Duration
}

func NewChatRepository(defaultTTL time.Duration) *chatRepository {
	return &chatRepository{
		sessions:   make(map[int64]sessionEntry),
		chatTTL:    make(map[int64]time.Duration),
		defaultTTL: defaultTTL,
	}
}

func (c *chatRepository) SetTTL(chatID int64, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chatTTL[chatID] = ttl
}

func (c *chatRepository) SaveSession(chatID int64, session domain.ChatSession) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get chat TTL
	ttl := c.defaultTTL
	if chatTTL, ok := c.chatTTL[chatID]; ok {
		ttl = chatTTL
	}

	// Clean up expired session before saving a new one
	if entry, ok := c.sessions[chatID]; ok {
		if ttl > 0 && time.Since(entry.lastUpdate) > ttl {
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
	defer c.mu.RUnlock()

	// Get chat TTL
	ttl := c.defaultTTL
	if chatTTL, ok := c.chatTTL[chatID]; ok {
		ttl = chatTTL
	}

	entry, ok := c.sessions[chatID]
	if !ok {
		return domain.ChatSession{}, false
	}

	if ttl > 0 && time.Since(entry.lastUpdate) > ttl {
		return domain.ChatSession{}, false
	}

	return entry.session, true
}

func (c *chatRepository) ClearChat(chatID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.sessions, chatID)
}
