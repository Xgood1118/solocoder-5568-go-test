package httpclient

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"apitester/internal/models"
)

type Client struct {
	HTTPClient *http.Client
}

type ClientOption struct {
	Timeout        time.Duration
	Proxy          string
	Insecure       bool
	FollowRedirect bool
}

func NewClient(opt *ClientOption) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}

	if opt != nil {
		if opt.Insecure {
			transport.TLSClientConfig.InsecureSkipVerify = true
		}

		if opt.Proxy != "" {
			proxyURL, err := url.Parse(opt.Proxy)
			if err == nil {
				transport.Proxy = http.ProxyURL(proxyURL)
			}
		}
	}

	client := &http.Client{
		Transport: transport,
	}

	if opt != nil {
		if opt.Timeout > 0 {
			client.Timeout = opt.Timeout
		}

		if !opt.FollowRedirect {
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
		}
	}

	return &Client{
		HTTPClient: client,
	}
}

func (c *Client) Do(req *models.Request) (*models.Response, error) {
	httpReq, err := BuildRequest(req)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	start := time.Now()
	httpResp, err := c.HTTPClient.Do(httpReq)
	latency := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return parseResponse(httpResp, latency)
}
