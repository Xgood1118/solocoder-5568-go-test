package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"

	"apitester/internal/models"
	"apitester/pkg/utils"
)

func BuildJSONBody(data any) (io.Reader, string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, "", fmt.Errorf("marshal json: %w", err)
	}
	return bytes.NewReader(b), "application/json", nil
}

func BuildFormBody(form map[string]string) (io.Reader, string, error) {
	values := url.Values{}
	for k, v := range form {
		values.Set(k, v)
	}
	return bytes.NewReader([]byte(values.Encode())), "application/x-www-form-urlencoded", nil
}

func BuildMultipartBody(fields []*models.MultipartField) (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for _, field := range fields {
		if field.File != "" {
			filePath := field.File
			fileContent, err := utils.ReadFile(filePath)
			if err != nil {
				return nil, "", fmt.Errorf("read file %s: %w", filePath, err)
			}

			contentType := field.ContentType
			if contentType == "" {
				contentType = http.DetectContentType(fileContent)
			}

			part, err := writer.CreateFormFile(field.Name, filepath.Base(filePath))
			if err != nil {
				return nil, "", fmt.Errorf("create form file %s: %w", field.Name, err)
			}

			if _, err := part.Write(fileContent); err != nil {
				return nil, "", fmt.Errorf("write file content %s: %w", field.Name, err)
			}
		} else {
			if err := writer.WriteField(field.Name, field.Value); err != nil {
				return nil, "", fmt.Errorf("write field %s: %w", field.Name, err)
			}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart writer: %w", err)
	}

	return &buf, writer.FormDataContentType(), nil
}

func BuildRawBody(raw string, contentType string) (io.Reader, string, error) {
	if contentType == "" {
		contentType = "text/plain"
	}
	return bytes.NewReader([]byte(raw)), contentType, nil
}

func BuildGraphQLBody(gql *models.GraphQLBody) (io.Reader, string, error) {
	body := map[string]any{
		"query": gql.Query,
	}
	if len(gql.Variables) > 0 {
		body["variables"] = gql.Variables
	}
	if gql.OperationName != "" {
		body["operationName"] = gql.OperationName
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, "", fmt.Errorf("marshal graphql body: %w", err)
	}
	return bytes.NewReader(b), "application/json", nil
}
