package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"apitester/internal/models"
)

type Loader struct{}

func NewLoader() *Loader {
	return &Loader{}
}

func (l *Loader) LoadYAML(path string) (*models.TestSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read yaml file: %w", err)
	}
	return l.LoadYAMLFromBytes(data)
}

func (l *Loader) LoadYAMLFromBytes(data []byte) (*models.TestSuite, error) {
	var suite models.TestSuite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}
	return &suite, nil
}

func (l *Loader) LoadJSON(path string) (*models.TestSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read json file: %w", err)
	}
	return l.LoadJSONFromBytes(data)
}

func (l *Loader) LoadJSONFromBytes(data []byte) (*models.TestSuite, error) {
	var suite models.TestSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}
	return &suite, nil
}

func (l *Loader) LoadAuto(path string) (*models.TestSuite, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return l.LoadYAML(path)
	case ".json":
		return l.LoadJSON(path)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}
}

func (l *Loader) ResolveRelativePath(basePath, relPath string) string {
	if filepath.IsAbs(relPath) {
		return relPath
	}
	return filepath.Join(filepath.Dir(basePath), relPath)
}
