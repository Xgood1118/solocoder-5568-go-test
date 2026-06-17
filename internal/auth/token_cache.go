package auth

import (
	"sync"
	"time"

	"apitester/internal/models"
)

type TokenCache struct {
	mu    sync.RWMutex
	cache map[string]*models.TokenCache
}

func NewTokenCache() *TokenCache {
	return &TokenCache{
		cache: make(map[string]*models.TokenCache),
	}
}

func (tc *TokenCache) GetToken(key string) (*models.TokenCache, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	token, ok := tc.cache[key]
	return token, ok
}

func (tc *TokenCache) SetToken(key string, token *models.TokenCache) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cache[key] = token
}

func (tc *TokenCache) IsExpired(key string, buffer time.Duration) bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	token, ok := tc.cache[key]
	if !ok {
		return true
	}
	return time.Now().Add(buffer).After(token.ExpiresAt)
}

func (tc *TokenCache) Delete(key string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	delete(tc.cache, key)
}

func (tc *TokenCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cache = make(map[string]*models.TokenCache)
}
