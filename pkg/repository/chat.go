package repository

import (
	"sync"
	"time"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type chatEntry struct {
	chat       domain.Chat
	lastUpdate time.Time
}

type chatRepository struct {
	mu         sync.RWMutex
	chats      map[int64]chatEntry
	chatTTL    map[int64]time.Duration
	defaultTTL time.Duration
}

func NewChatRepository(defaultTTL time.Duration) *chatRepository {
	return &chatRepository{
		chats:      make(map[int64]chatEntry),
		chatTTL:    make(map[int64]time.Duration),
		defaultTTL: defaultTTL,
	}
}

func (c *chatRepository) SetTTL(chatID int64, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chatTTL[chatID] = ttl
}

func (c *chatRepository) Save(chat domain.Chat) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get chat TTL
	ttl := c.defaultTTL
	if chatTTL, ok := c.chatTTL[chat.ID]; ok {
		ttl = chatTTL
	}

	// Clean up expired chat before saving a new one
	if entry, ok := c.chats[chat.ID]; ok {
		if ttl > 0 && time.Since(entry.lastUpdate) > ttl {
			delete(c.chats, chat.ID)
		}
	}

	c.chats[chat.ID] = chatEntry{
		chat:       chat,
		lastUpdate: time.Now(),
	}
}

func (c *chatRepository) GetByID(chatID int64) (domain.Chat, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Get chat TTL
	ttl := c.defaultTTL
	if chatTTL, ok := c.chatTTL[chatID]; ok {
		ttl = chatTTL
	}

	entry, ok := c.chats[chatID]
	if !ok {
		return domain.Chat{}, false
	}

	if ttl > 0 && time.Since(entry.lastUpdate) > ttl {
		return domain.Chat{}, false
	}

	return entry.chat, true
}

func (c *chatRepository) ClearChat(chatID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.chats, chatID)
}
