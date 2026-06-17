package providers

import (
	"fmt"
	"net/http"
	"time"

	"apitester/internal/models"
)

type OAuth2Provider struct {
	config *models.OAuth2Auth
}

func NewOAuth2Provider(config *models.OAuth2Auth) *OAuth2Provider {
	return &OAuth2Provider{config: config}
}

func (p *OAuth2Provider) Apply(req *http.Request, tokenCache TokenCache, refresher TokenRefresher, cacheKey string) error {
	cached, ok := tokenCache.GetToken(cacheKey)
	if !ok || tokenCache.IsExpired(cacheKey, 60*time.Second) {
		config := *p.config

		if ok && cached.RefreshToken != "" && !tokenCache.IsExpired(cacheKey, 0) {
			config.GrantType = "refresh_token"
			config.RefreshToken = cached.RefreshToken
		}

		newToken, err := refresher.RefreshOAuth2Token(&config)
		if err != nil {
			if ok && !tokenCache.IsExpired(cacheKey, 0) {
				req.Header.Set("Authorization", "Bearer "+cached.Token)
				return nil
			}
			return fmt.Errorf("failed to refresh oauth2 token: %w", err)
		}

		tokenCache.SetToken(cacheKey, newToken)
		req.Header.Set("Authorization", "Bearer "+newToken.Token)
		return nil
	}

	req.Header.Set("Authorization", "Bearer "+cached.Token)
	return nil
}
