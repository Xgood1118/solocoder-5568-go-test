package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"apitester/internal/auth/providers"
	"apitester/internal/models"
)

type AuthManager struct {
	tokenCache *TokenCache
	mu         sync.RWMutex
	providers  map[string]providers.AuthProvider
}

func NewAuthManager() *AuthManager {
	return &AuthManager{
		tokenCache: NewTokenCache(),
		providers:  make(map[string]providers.AuthProvider),
	}
}

func (am *AuthManager) ApplyAuth(req *http.Request, authConfig *models.AuthConfig) error {
	if authConfig == nil {
		return nil
	}

	provider, err := am.getOrCreateProvider(authConfig)
	if err != nil {
		return err
	}

	cacheKey := am.generateCacheKey(authConfig)
	return provider.Apply(req, am.tokenCache, am, cacheKey)
}

func (am *AuthManager) RefreshTokenIfNeeded(authConfig *models.AuthConfig) error {
	if authConfig == nil {
		return nil
	}

	cacheKey := am.generateCacheKey(authConfig)

	if !am.tokenCache.IsExpired(cacheKey, 60*time.Second) {
		return nil
	}

	switch authConfig.Type {
	case "bearer":
		config, err := am.toBearerAuth(authConfig.Config)
		if err != nil {
			return err
		}
		if config.TokenURL == "" {
			return nil
		}
		newToken, err := am.RefreshBearerToken(config)
		if err != nil {
			return fmt.Errorf("failed to refresh bearer token: %w", err)
		}
		am.tokenCache.SetToken(cacheKey, newToken)

	case "oauth2":
		config, err := am.toOAuth2Auth(authConfig.Config)
		if err != nil {
			return err
		}
		cached, ok := am.tokenCache.GetToken(cacheKey)
		configCopy := *config
		if ok && cached.RefreshToken != "" {
			configCopy.GrantType = "refresh_token"
			configCopy.RefreshToken = cached.RefreshToken
		}
		newToken, err := am.RefreshOAuth2Token(&configCopy)
		if err != nil {
			return fmt.Errorf("failed to refresh oauth2 token: %w", err)
		}
		am.tokenCache.SetToken(cacheKey, newToken)
	}

	return nil
}

func (am *AuthManager) getOrCreateProvider(authConfig *models.AuthConfig) (providers.AuthProvider, error) {
	am.mu.RLock()
	provider, ok := am.providers[authConfig.Type]
	am.mu.RUnlock()

	if ok {
		return provider, nil
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	if provider, ok := am.providers[authConfig.Type]; ok {
		return provider, nil
	}

	var err error
	switch authConfig.Type {
	case "bearer":
		var config *models.BearerAuth
		config, err = am.toBearerAuth(authConfig.Config)
		if err == nil {
			provider = providers.NewBearerProvider(config)
		}
	case "basic":
		var config *models.BasicAuth
		config, err = am.toBasicAuth(authConfig.Config)
		if err == nil {
			provider = providers.NewBasicProvider(config)
		}
	case "digest":
		var config *models.DigestAuth
		config, err = am.toDigestAuth(authConfig.Config)
		if err == nil {
			provider = providers.NewDigestProvider(config)
		}
	case "oauth2":
		var config *models.OAuth2Auth
		config, err = am.toOAuth2Auth(authConfig.Config)
		if err == nil {
			provider = providers.NewOAuth2Provider(config)
		}
	case "apikey":
		var config *models.APIKeyAuth
		config, err = am.toAPIKeyAuth(authConfig.Config)
		if err == nil {
			provider = providers.NewAPIKeyProvider(config)
		}
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", authConfig.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create %s provider: %w", authConfig.Type, err)
	}

	am.providers[authConfig.Type] = provider
	return provider, nil
}

func (am *AuthManager) generateCacheKey(authConfig *models.AuthConfig) string {
	configBytes, _ := json.Marshal(authConfig.Config)
	return fmt.Sprintf("%s:%s", authConfig.Type, string(configBytes))
}

func (am *AuthManager) toBearerAuth(config interface{}) (*models.BearerAuth, error) {
	if c, ok := config.(*models.BearerAuth); ok {
		return c, nil
	}
	if c, ok := config.(models.BearerAuth); ok {
		return &c, nil
	}

	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("invalid bearer auth config: %w", err)
	}

	var result models.BearerAuth
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid bearer auth config: %w", err)
	}
	return &result, nil
}

func (am *AuthManager) toBasicAuth(config interface{}) (*models.BasicAuth, error) {
	if c, ok := config.(*models.BasicAuth); ok {
		return c, nil
	}
	if c, ok := config.(models.BasicAuth); ok {
		return &c, nil
	}

	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("invalid basic auth config: %w", err)
	}

	var result models.BasicAuth
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid basic auth config: %w", err)
	}
	return &result, nil
}

func (am *AuthManager) toDigestAuth(config interface{}) (*models.DigestAuth, error) {
	if c, ok := config.(*models.DigestAuth); ok {
		return c, nil
	}
	if c, ok := config.(models.DigestAuth); ok {
		return &c, nil
	}

	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("invalid digest auth config: %w", err)
	}

	var result models.DigestAuth
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid digest auth config: %w", err)
	}
	return &result, nil
}

func (am *AuthManager) toOAuth2Auth(config interface{}) (*models.OAuth2Auth, error) {
	if c, ok := config.(*models.OAuth2Auth); ok {
		return c, nil
	}
	if c, ok := config.(models.OAuth2Auth); ok {
		return &c, nil
	}

	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("invalid oauth2 auth config: %w", err)
	}

	var result models.OAuth2Auth
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid oauth2 auth config: %w", err)
	}
	return &result, nil
}

func (am *AuthManager) toAPIKeyAuth(config interface{}) (*models.APIKeyAuth, error) {
	if c, ok := config.(*models.APIKeyAuth); ok {
		return c, nil
	}
	if c, ok := config.(models.APIKeyAuth); ok {
		return &c, nil
	}

	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("invalid apikey auth config: %w", err)
	}

	var result models.APIKeyAuth
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid apikey auth config: %w", err)
	}
	return &result, nil
}

func (am *AuthManager) ClearCache() {
	am.tokenCache.Clear()
}

func (am *AuthManager) GetTokenCache() *TokenCache {
	return am.tokenCache
}

func (am *AuthManager) RefreshBearerToken(config *models.BearerAuth) (*models.TokenCache, error) {
	return RefreshBearerToken(config)
}

func (am *AuthManager) RefreshOAuth2Token(config *models.OAuth2Auth) (*models.TokenCache, error) {
	return RefreshOAuth2Token(config)
}
