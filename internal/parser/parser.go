package parser

import (
	"fmt"

	"apitester/internal/models"
)

type Parser struct {
	loader    *Loader
	validator *Validator
	resolver  *Resolver
}

func NewParser() *Parser {
	loader := NewLoader()
	return &Parser{
		loader:    loader,
		validator: NewValidator(),
		resolver:  NewResolver(loader),
	}
}

func (p *Parser) LoadSuite(path string) (*models.TestSuite, error) {
	suite, err := p.loader.LoadAuto(path)
	if err != nil {
		return nil, fmt.Errorf("load suite: %w", err)
	}

	suite, err = p.resolver.ResolveIncludes(suite, path)
	if err != nil {
		return nil, fmt.Errorf("resolve includes: %w", err)
	}

	if err := p.validator.ValidateSuite(suite); err != nil {
		return nil, fmt.Errorf("validate suite: %w", err)
	}

	return suite, nil
}

func (p *Parser) LoadSuiteFromBytes(data []byte, format string) (*models.TestSuite, error) {
	var suite *models.TestSuite
	var err error

	switch format {
	case "yaml", "yml":
		suite, err = p.loader.LoadYAMLFromBytes(data)
	case "json":
		suite, err = p.loader.LoadJSONFromBytes(data)
	default:
		return nil, fmt.Errorf("unsupported format: %s, must be 'yaml' or 'json'", format)
	}

	if err != nil {
		return nil, fmt.Errorf("load suite from bytes: %w", err)
	}

	if err := p.validator.ValidateSuite(suite); err != nil {
		return nil, fmt.Errorf("validate suite: %w", err)
	}

	return suite, nil
}
