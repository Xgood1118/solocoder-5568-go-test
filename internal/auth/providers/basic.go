package providers

import (
	"encoding/base64"
	"net/http"

	"apitester/internal/models"
)

type BasicProvider struct {
	config *models.BasicAuth
}

func NewBasicProvider(config *models.BasicAuth) *BasicProvider {
	return &BasicProvider{config: config}
}

func (p *BasicProvider) Apply(req *http.Request, tokenCache TokenCache, refresher TokenRefresher, cacheKey string) error {
	authStr := p.config.Username + ":" + p.config.Password
	encoded := base64.StdEncoding.EncodeToString([]byte(authStr))
	req.Header.Set("Authorization", "Basic "+encoded)
	return nil
}
