package providers

import (
	"fmt"
	"net/http"
	"time"

	"apitester/internal/models"
)

type BearerProvider struct {
	config *models.BearerAuth
}

func NewBearerProvider(config *models.BearerAuth) *BearerProvider {
	return &BearerProvider{config: config}
}

func (p *BearerProvider) Apply(req *http.Request, tokenCache TokenCache, refresher TokenRefresher, cacheKey string) error {
	token := p.config.Token

	if p.config.TokenURL != "" {
		cached, ok := tokenCache.GetToken(cacheKey)
		if !ok || tokenCache.IsExpired(cacheKey, 60*time.Second) {
			newToken, err := refresher.RefreshBearerToken(p.config)
			if err != nil {
				if ok && !tokenCache.IsExpired(cacheKey, 0) {
					token = cached.Token
				} else {
					return fmt.Errorf("failed to refresh bearer token: %w", err)
				}
			} else {
				tokenCache.SetToken(cacheKey, newToken)
				token = newToken.Token
			}
		} else {
			token = cached.Token
		}
	}

	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}
