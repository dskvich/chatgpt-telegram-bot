package repository

import (
	"fmt"
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
	chats map[string]chatEntry
}

func NewChatRepository() *chatRepository {
	return &chatRepository{
		chats: make(map[string]chatEntry),
	}
}

func (c *chatRepository) key(chatID int64, topicID int) string {
	return fmt.Sprintf("%d:%d", chatID, topicID)
}

func (c *chatRepository) Save(chat domain.Chat) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.key(chat.ID, chat.TopicID)
	c.chats[key] = chatEntry{
		chat:       chat,
		lastUpdate: time.Now(),
	}
}

func (c *chatRepository) Get(chatID int64, topicID int) (domain.Chat, time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.key(chatID, topicID)
	entry, ok := c.chats[key]
	if !ok {
		return domain.Chat{}, time.Time{}, false
	}

	return entry.chat, entry.lastUpdate, true
}

func (c *chatRepository) Clear(chatID int64, topicID int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.key(chatID, topicID)
	delete(c.chats, key)
}
