package openapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"apitester/internal/models"

	"github.com/getkin/kin-openapi/openapi3"
)

type TestCaseGenerator struct {
	spec *openapi3.T
}

func NewTestCaseGenerator(spec *openapi3.T) *TestCaseGenerator {
	return &TestCaseGenerator{
		spec: spec,
	}
}

func (g *TestCaseGenerator) GenerateSuite() *models.TestSuite {
	suite := &models.TestSuite{
		Name:        g.getSuiteName(),
		Description: g.getSuiteDescription(),
		Version:     g.spec.Info.Version,
		BaseURL:     g.getBaseURL(),
		TestCases:   make([]*models.TestCase, 0),
		Variables:   make(map[string]any),
	}

	if g.spec.Info != nil && g.spec.Info.Contact != nil {
		if g.spec.Info.Contact.Name != "" {
			suite.Variables["contact_name"] = g.spec.Info.Contact.Name
		}
		if g.spec.Info.Contact.Email != "" {
			suite.Variables["contact_email"] = g.spec.Info.Contact.Email
		}
	}

	for path, pathItem := range g.spec.Paths.Map() {
		if pathItem == nil {
			continue
		}

		operations := map[string]*openapi3.Operation{
			http.MethodGet:     pathItem.Get,
			http.MethodPost:    pathItem.Post,
			http.MethodPut:     pathItem.Put,
			http.MethodDelete:  pathItem.Delete,
			http.MethodPatch:   pathItem.Patch,
			http.MethodHead:    pathItem.Head,
			http.MethodOptions: pathItem.Options,
			http.MethodTrace:   pathItem.Trace,
		}

		for method, operation := range operations {
			if operation == nil {
				continue
			}

			testCase := g.GenerateCase(path, method, operation, pathItem)
			suite.TestCases = append(suite.TestCases, testCase)
		}
	}

	return suite
}

func (g *TestCaseGenerator) GenerateCase(path, method string, operation *openapi3.Operation, pathItem *openapi3.PathItem) *models.TestCase {
	testCase := &models.TestCase{
		Name:        g.getCaseName(path, method, operation),
		Description: operation.Description,
		Tags:        operation.Tags,
		Request: &models.Request{
			Method:         method,
			URL:            path,
			Headers:        make(map[string]string),
			QueryParams:    make(map[string]string),
			FollowRedirect: true,
		},
		Assertions: make([]*models.Assertion, 0),
		Extract:    make(map[string]string),
	}

	if operation.OperationID != "" {
		testCase.ID = operation.OperationID
	}

	allParams := make([]*openapi3.ParameterRef, 0)
	if pathItem != nil {
		allParams = append(allParams, pathItem.Parameters...)
	}
	allParams = append(allParams, operation.Parameters...)

	pathParams, queryParams, headerParams := g.GenerateParameters(allParams)

	for name, value := range pathParams {
		placeholder := fmt.Sprintf("{%s}", name)
		testCase.Request.URL = strings.ReplaceAll(testCase.Request.URL, placeholder, value)
	}

	for name, value := range queryParams {
		testCase.Request.QueryParams[name] = value
	}

	for name, value := range headerParams {
		testCase.Request.Headers[name] = value
	}

	if operation.RequestBody != nil && operation.RequestBody.Value != nil {
		body, contentType := g.GenerateRequestBody(operation.RequestBody.Value)
		if body != nil {
			testCase.Request.Body = body
			if contentType != "" {
				testCase.Request.Headers["Content-Type"] = contentType
			}
		}
	}

	g.GenerateAssertions(testCase, operation)

	return testCase
}

func (g *TestCaseGenerator) GenerateParameters(params []*openapi3.ParameterRef) (pathParams, queryParams, headerParams map[string]string) {
	pathParams = make(map[string]string)
	queryParams = make(map[string]string)
	headerParams = make(map[string]string)

	for _, paramRef := range params {
		if paramRef == nil || paramRef.Value == nil {
			continue
		}

		param := paramRef.Value
		var exampleValue string

		if param.Example != nil {
			exampleValue = fmt.Sprintf("%v", param.Example)
		} else if param.Schema != nil && param.Schema.Value != nil {
			example := GenerateExampleFromSchema(param.Schema.Value)
			exampleValue = formatValue(example)
		} else {
			exampleValue = g.getDefaultValueForType(param.Schema)
		}

		switch param.In {
		case "path":
			pathParams[param.Name] = exampleValue
		case "query":
			queryParams[param.Name] = exampleValue
		case "header":
			if !strings.EqualFold(param.Name, "Content-Type") && !strings.EqualFold(param.Name, "Accept") {
				headerParams[param.Name] = exampleValue
			}
		}
	}

	return pathParams, queryParams, headerParams
}

func (g *TestCaseGenerator) GenerateRequestBody(requestBody *openapi3.RequestBody) (*models.RequestBody, string) {
	if requestBody == nil || requestBody.Content == nil {
		return nil, ""
	}

	contentTypes := []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
		"application/xml",
		"text/plain",
	}

	for _, contentType := range contentTypes {
		mediaType, ok := requestBody.Content[contentType]
		if !ok || mediaType == nil {
			continue
		}

		body := &models.RequestBody{
			ContentType: contentType,
		}

		switch contentType {
		case "application/json":
			body.Type = "json"
			if mediaType.Example != nil {
				body.JSON = mediaType.Example
			} else if mediaType.Schema != nil && mediaType.Schema.Value != nil {
				body.JSON = GenerateExampleFromSchema(mediaType.Schema.Value)
			} else {
				body.JSON = map[string]any{}
			}
			return body, contentType

		case "application/x-www-form-urlencoded":
			body.Type = "form"
			body.Form = make(map[string]string)
			if mediaType.Schema != nil && mediaType.Schema.Value != nil {
				example := GenerateExampleFromSchema(mediaType.Schema.Value)
				if m, ok := example.(map[string]any); ok {
					for k, v := range m {
						body.Form[k] = formatValue(v)
					}
				}
			}
			return body, contentType

		case "multipart/form-data":
			body.Type = "multipart"
			body.Multipart = make([]*models.MultipartField, 0)
			if mediaType.Schema != nil && mediaType.Schema.Value != nil {
				example := GenerateExampleFromSchema(mediaType.Schema.Value)
				if m, ok := example.(map[string]any); ok {
					for k, v := range m {
						body.Multipart = append(body.Multipart, &models.MultipartField{
							Name:  k,
							Value: formatValue(v),
						})
					}
				}
			}
			return body, contentType

		case "application/xml", "text/plain":
			body.Type = "raw"
			if mediaType.Example != nil {
				body.Raw = formatValue(mediaType.Example)
			} else {
				body.Raw = g.getDefaultValueForType(mediaType.Schema)
			}
			return body, contentType
		}
	}

	for contentType, mediaType := range requestBody.Content {
		if mediaType == nil {
			continue
		}
		body := &models.RequestBody{
			Type:        "raw",
			ContentType: contentType,
		}
		if mediaType.Example != nil {
			body.Raw = formatValue(mediaType.Example)
		}
		return body, contentType
	}

	return nil, ""
}

func (g *TestCaseGenerator) GenerateAssertions(testCase *models.TestCase, operation *openapi3.Operation) {
	if operation.Responses == nil {
		testCase.Assertions = append(testCase.Assertions, &models.Assertion{
			Type:     "status_code",
			Operator: "between",
			Value:    []int{200, 299},
			Message:  "Status code should be in 2xx range",
		})
		return
	}

	successStatus := 0
	for statusStr := range operation.Responses.Map() {
		if statusStr == "default" {
			continue
		}
		status, err := strconv.Atoi(statusStr)
		if err == nil && status >= 200 && status < 300 {
			successStatus = status
			break
		}
	}

	if successStatus > 0 {
		testCase.Assertions = append(testCase.Assertions, &models.Assertion{
			Type:     "status_code",
			Operator: "equals",
			Value:    successStatus,
			Message:  fmt.Sprintf("Status code should be %d", successStatus),
		})
	} else {
		testCase.Assertions = append(testCase.Assertions, &models.Assertion{
			Type:     "status_code",
			Operator: "between",
			Value:    []int{200, 299},
			Message:  "Status code should be in 2xx range",
		})
	}

	testCase.Assertions = append(testCase.Assertions, &models.Assertion{
		Type:     "header",
		Property: "Content-Type",
		Operator: "contains",
		Value:    "application/json",
		Message:  "Content-Type should contain application/json",
	})

	for statusStr, responseRef := range operation.Responses.Map() {
		if responseRef == nil || responseRef.Value == nil {
			continue
		}

		status, err := strconv.Atoi(statusStr)
		if err != nil || status < 200 || status >= 300 {
			continue
		}

		response := responseRef.Value
		if response.Content == nil {
			continue
		}

		if mediaType, ok := response.Content["application/json"]; ok && mediaType != nil {
			if mediaType.Schema != nil && mediaType.Schema.Value != nil {
				g.addJSONAssertions(testCase, mediaType.Schema.Value, "")
			}
		}
		break
	}
}

func (g *TestCaseGenerator) addJSONAssertions(testCase *models.TestCase, schema *openapi3.Schema, prefix string) {
	if schema == nil {
		return
	}

	if (schema.Type != nil && schema.Type.Is("object")) || ((schema.Type == nil || schema.Type.IsEmpty()) && len(schema.Properties) > 0) {
		for _, required := range schema.Required {
			jsonPath := fmt.Sprintf("$.%s", required)
			if prefix != "" {
				jsonPath = fmt.Sprintf("%s.%s", prefix, required)
			} else {
				jsonPath = fmt.Sprintf("$.%s", required)
			}

			testCase.Assertions = append(testCase.Assertions, &models.Assertion{
				Type:     "json_path",
				Property: jsonPath,
				Operator: "not_null",
				Message:  fmt.Sprintf("Field '%s' should exist and not be null", required),
			})

			if propRef, ok := schema.Properties[required]; ok && propRef != nil && propRef.Value != nil {
				prop := propRef.Value
				if (prop.Type != nil && prop.Type.Is("object")) || ((prop.Type == nil || prop.Type.IsEmpty()) && len(prop.Properties) > 0) {
					var newPrefix string
					if prefix != "" {
						newPrefix = fmt.Sprintf("%s.%s", prefix, required)
					} else {
						newPrefix = fmt.Sprintf("$.%s", required)
					}
					g.addJSONAssertions(testCase, prop, newPrefix)
				}
			}
		}
	} else if schema.Type != nil && schema.Type.Is("array") && schema.Items != nil && schema.Items.Value != nil {
		itemSchema := schema.Items.Value
		if (itemSchema.Type != nil && itemSchema.Type.Is("object")) || ((itemSchema.Type == nil || itemSchema.Type.IsEmpty()) && len(itemSchema.Properties) > 0) {
			var newPrefix string
			if prefix != "" {
				newPrefix = fmt.Sprintf("%s[0]", prefix)
			} else {
				newPrefix = "$[0]"
			}
			g.addJSONAssertions(testCase, itemSchema, newPrefix)
		}
	}
}

func (g *TestCaseGenerator) getSuiteName() string {
	if g.spec.Info != nil && g.spec.Info.Title != "" {
		return g.spec.Info.Title + " Test Suite"
	}
	return "Generated Test Suite"
}

func (g *TestCaseGenerator) getSuiteDescription() string {
	if g.spec.Info != nil && g.spec.Info.Description != "" {
		return g.spec.Info.Description
	}
	return "Auto-generated test suite from OpenAPI specification"
}

func (g *TestCaseGenerator) getBaseURL() string {
	if g.spec.Servers != nil && len(g.spec.Servers) > 0 {
		for _, server := range g.spec.Servers {
			if server != nil && server.URL != "" {
				return server.URL
			}
		}
	}
	return "{{base_url}}"
}

func (g *TestCaseGenerator) getCaseName(path, method string, operation *openapi3.Operation) string {
	if operation.Summary != "" {
		return operation.Summary
	}
	if operation.OperationID != "" {
		return operation.OperationID
	}
	return fmt.Sprintf("%s %s", strings.ToUpper(method), path)
}

func (g *TestCaseGenerator) getDefaultValueForType(schemaRef *openapi3.SchemaRef) string {
	if schemaRef != nil && schemaRef.Value != nil {
		schema := schemaRef.Value
		switch {
		case schema.Type != nil && schema.Type.Is("string"):
			return "string_value"
		case schema.Type != nil && schema.Type.Is("number"):
			return "123.45"
		case schema.Type != nil && schema.Type.Is("integer"):
			return "123"
		case schema.Type != nil && schema.Type.Is("boolean"):
			return "true"
		default:
			return "value"
		}
	}
	return "value"
}

func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
