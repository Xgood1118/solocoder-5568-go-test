package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"apitester/internal/models"
	"apitester/pkg/utils"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

const (
	maxRetries     = 3
	baseBackoff    = 1 * time.Second
	maxBackoff     = 10 * time.Second
	refreshBuffer  = 60 * time.Second
)

func RefreshBearerToken(config *models.BearerAuth) (*models.TokenCache, error) {
	if config.TokenURL == "" {
		return nil, fmt.Errorf("token_url is required for token refresh")
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := utils.ExponentialBackoff(attempt-1, baseBackoff, maxBackoff)
			time.Sleep(backoff)
		}

		token, err := fetchBearerToken(config)
		if err != nil {
			lastErr = err
			continue
		}
		return token, nil
	}
	return nil, fmt.Errorf("failed to refresh bearer token after %d attempts: %w", maxRetries, lastErr)
}

func fetchBearerToken(config *models.BearerAuth) (*models.TokenCache, error) {
	form := url.Values{}
	if config.ClientID != "" {
		form.Set("client_id", config.ClientID)
	}
	if config.ClientSecret != "" {
		form.Set("client_secret", config.ClientSecret)
	}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", config.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	expiresIn := config.ExpiresIn
	if tokenResp.ExpiresIn > 0 {
		expiresIn = tokenResp.ExpiresIn
	}
	if expiresIn == 0 {
		expiresIn = 3600
	}

	return &models.TokenCache{
		Token:        tokenResp.AccessToken,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
		RefreshToken: tokenResp.RefreshToken,
	}, nil
}

func RefreshOAuth2Token(config *models.OAuth2Auth) (*models.TokenCache, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := utils.ExponentialBackoff(attempt-1, baseBackoff, maxBackoff)
			time.Sleep(backoff)
		}

		token, err := fetchOAuth2Token(config)
		if err != nil {
			lastErr = err
			continue
		}
		return token, nil
	}
	return nil, fmt.Errorf("failed to refresh oauth2 token after %d attempts: %w", maxRetries, lastErr)
}

func fetchOAuth2Token(config *models.OAuth2Auth) (*models.TokenCache, error) {
	form := url.Values{}
	form.Set("grant_type", config.GrantType)
	form.Set("client_id", config.ClientID)
	form.Set("client_secret", config.ClientSecret)

	switch config.GrantType {
	case "password":
		form.Set("username", config.Username)
		form.Set("password", config.Password)
	case "refresh_token":
		if config.RefreshToken != "" {
			form.Set("refresh_token", config.RefreshToken)
		}
	case "client_credentials":
	default:
	}

	if len(config.Scopes) > 0 {
		form.Set("scope", strings.Join(config.Scopes, " "))
	}

	for k, v := range config.Params {
		form.Set(k, v)
	}

	reqBody := bytes.NewBufferString(form.Encode())
	req, err := http.NewRequest("POST", config.TokenURL, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	expiresIn := config.ExpiresIn
	if tokenResp.ExpiresIn > 0 {
		expiresIn = tokenResp.ExpiresIn
	}
	if expiresIn == 0 {
		expiresIn = 3600
	}

	newRefreshToken := config.RefreshToken
	if tokenResp.RefreshToken != "" {
		newRefreshToken = tokenResp.RefreshToken
	}

	return &models.TokenCache{
		Token:        tokenResp.AccessToken,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
		RefreshToken: newRefreshToken,
	}, nil
}
