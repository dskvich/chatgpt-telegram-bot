package repository

import (
	"sync"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type stateRepository struct {
	mu    sync.RWMutex
	state map[int64]domain.State
}

func NewStateRepository() *stateRepository {
	return &stateRepository{
		state: make(map[int64]domain.State),
	}
}

func (s *stateRepository) Save(chatID int64, state domain.State) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state[chatID] = state
}

func (s *stateRepository) GetByChatID(chatID int64) (domain.State, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.state[chatID]
	return state, exists
}

func (s *stateRepository) Clear(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.state, chatID)
}
