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
	mu    sync.RWMutex
	chats map[int64]chatEntry
}

func NewChatRepository() *chatRepository {
	return &chatRepository{
		chats: make(map[int64]chatEntry),
	}
}

func (c *chatRepository) Save(chat domain.Chat) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chats[chat.ID] = chatEntry{
		chat:       chat,
		lastUpdate: time.Now(),
	}
}

func (c *chatRepository) GetByID(chatID int64) (domain.Chat, time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.chats[chatID]
	if !ok {
		return domain.Chat{}, time.Time{}, false
	}

	return entry.chat, entry.lastUpdate, true
}

func (c *chatRepository) Clear(chatID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.chats, chatID)
}
