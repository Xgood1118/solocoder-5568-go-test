package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"apitester/internal/models"
)

func BuildRequest(req *models.Request) (*http.Request, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	if req.Method == "" {
		return nil, fmt.Errorf("request method is empty")
	}
	if req.URL == "" {
		return nil, fmt.Errorf("request url is empty")
	}

	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, fmt.Errorf("parse url %s: %w", req.URL, err)
	}

	if len(req.QueryParams) > 0 {
		q := u.Query()
		for k, v := range req.QueryParams {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	var body io.Reader
	var contentType string

	if req.Body != nil {
		switch req.Body.Type {
		case "json":
			body, contentType, err = BuildJSONBody(req.Body.JSON)
		case "form":
			body, contentType, err = BuildFormBody(req.Body.Form)
		case "multipart":
			body, contentType, err = BuildMultipartBody(req.Body.Multipart)
		case "raw":
			body, contentType, err = BuildRawBody(req.Body.Raw, req.Body.ContentType)
		case "graphql":
			if req.Body.GraphQL == nil {
				return nil, fmt.Errorf("graphql body is nil")
			}
			body, contentType, err = BuildGraphQLBody(req.Body.GraphQL)
		case "":
		default:
			return nil, fmt.Errorf("unknown body type: %s", req.Body.Type)
		}
		if err != nil {
			return nil, err
		}
	}

	httpReq, err := http.NewRequest(req.Method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	if contentType != "" && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	return httpReq, nil
}
