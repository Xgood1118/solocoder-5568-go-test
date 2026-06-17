package providers

import (
	"net/http"
	"time"

	"apitester/internal/models"
)

type TokenCache interface {
	GetToken(key string) (*models.TokenCache, bool)
	SetToken(key string, token *models.TokenCache)
	IsExpired(key string, buffer time.Duration) bool
}

type TokenRefresher interface {
	RefreshBearerToken(config *models.BearerAuth) (*models.TokenCache, error)
	RefreshOAuth2Token(config *models.OAuth2Auth) (*models.TokenCache, error)
}

type AuthProvider interface {
	Apply(req *http.Request, tokenCache TokenCache, refresher TokenRefresher, cacheKey string) error
}
