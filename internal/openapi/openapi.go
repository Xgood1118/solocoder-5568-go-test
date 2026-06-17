package openapi

import (
	"fmt"

	"apitester/internal/models"

	"github.com/getkin/kin-openapi/openapi3"
)

type OpenAPIGenerator struct {
	spec *openapi3.T
}

func NewOpenAPIGenerator() *OpenAPIGenerator {
	return &OpenAPIGenerator{}
}

func (g *OpenAPIGenerator) GenerateFromFile(filePath string) (*models.TestSuite, error) {
	spec, err := ParseOpenAPIFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI file: %w", err)
	}
	g.spec = spec
	return g.GenerateFromSpec(spec)
}

func (g *OpenAPIGenerator) GenerateFromURL(url string) (*models.TestSuite, error) {
	spec, err := ParseOpenAPIURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI from URL: %w", err)
	}
	g.spec = spec
	return g.GenerateFromSpec(spec)
}

func (g *OpenAPIGenerator) GenerateFromSpec(spec *openapi3.T) (*models.TestSuite, error) {
	g.spec = spec

	generator := NewTestCaseGenerator(spec)
	suite := generator.GenerateSuite()

	return suite, nil
}
