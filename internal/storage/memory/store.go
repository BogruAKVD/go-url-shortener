package memory

import (
	"sync"

	"github.com/bogru/go-url-shortener/internal/shortener"
)

type Store struct {
	mu      sync.RWMutex
	urlByID map[string]string
}

func New() *Store {
	return &Store{
		urlByID: make(map[string]string),
	}
}

func (s *Store) Get(shortCode string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	originalURL, ok := s.urlByID[shortCode]
	if !ok {
		return "", shortener.ErrShortCodeMiss
	}

	return originalURL, nil
}

func (s *Store) Save(shortCode, originalURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.urlByID[shortCode]; exists {
		return shortener.ErrShortCodeBusy
	}

	s.urlByID[shortCode] = originalURL
	return nil
}
