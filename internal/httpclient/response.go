package httpclient

import (
	"io"
	"net/http"
	"time"

	"apitester/internal/models"
)

func parseResponse(resp *http.Response, latency time.Duration) (*models.Response, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	result := &models.Response{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
		BodyString: string(body),
		Latency:    latency,
		Protocol:   resp.Proto,
		TLS:        resp.TLS != nil,
	}

	return result, nil
}
