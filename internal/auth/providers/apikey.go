package providers

import (
	"fmt"
	"net/http"

	"apitester/internal/models"
)

type APIKeyProvider struct {
	config *models.APIKeyAuth
}

func NewAPIKeyProvider(config *models.APIKeyAuth) *APIKeyProvider {
	return &APIKeyProvider{config: config}
}

func (p *APIKeyProvider) Apply(req *http.Request, tokenCache TokenCache, refresher TokenRefresher, cacheKey string) error {
	switch p.config.In {
	case "header":
		req.Header.Set(p.config.Key, p.config.Value)
	case "query":
		q := req.URL.Query()
		q.Set(p.config.Key, p.config.Value)
		req.URL.RawQuery = q.Encode()
	default:
		return fmt.Errorf("unsupported api key location: %s, must be 'header' or 'query'", p.config.In)
	}
	return nil
}
