package parser

import (
	"fmt"
	"path/filepath"

	"apitester/internal/models"
)

type Resolver struct {
	loader *Loader
}

func NewResolver(loader *Loader) *Resolver {
	return &Resolver{
		loader: loader,
	}
}

type resolveContext struct {
	visited map[string]bool
	path    string
}

func newResolveContext() *resolveContext {
	return &resolveContext{
		visited: make(map[string]bool),
	}
}

func (r *Resolver) ResolveIncludes(suite *models.TestSuite, suitePath string) (*models.TestSuite, error) {
	ctx := newResolveContext()
	return r.resolveIncludesRecursive(suite, suitePath, ctx)
}

func (r *Resolver) resolveIncludesRecursive(suite *models.TestSuite, suitePath string, ctx *resolveContext) (*models.TestSuite, error) {
	absPath, err := filepath.Abs(suitePath)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}

	if ctx.visited[absPath] {
		return nil, fmt.Errorf("circular include detected: %s", absPath)
	}
	ctx.visited[absPath] = true
	ctx.path = absPath

	if len(suite.Includes) == 0 {
		return suite, nil
	}

	result := &models.TestSuite{}
	*result = *suite

	for _, includePath := range suite.Includes {
		resolvedPath := r.loader.ResolveRelativePath(absPath, includePath)

		includedSuite, err := r.loader.LoadAuto(resolvedPath)
		if err != nil {
			return nil, fmt.Errorf("load include '%s': %w", includePath, err)
		}

		includedSuite, err = r.resolveIncludesRecursive(includedSuite, resolvedPath, ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve includes for '%s': %w", includePath, err)
		}

		result = r.mergeSuites(includedSuite, result)
	}

	result.Includes = nil

	return result, nil
}

func (r *Resolver) mergeSuites(parent, child *models.TestSuite) *models.TestSuite {
	result := &models.TestSuite{}
	*result = *parent

	if child.Name != "" {
		result.Name = child.Name
	}
	if child.Description != "" {
		result.Description = child.Description
	}
	if child.Version != "" {
		result.Version = child.Version
	}

	result.Variables = deepMergeMaps(parent.Variables, child.Variables)

	if child.Auth != nil {
		result.Auth = child.Auth
	}

	if child.BaseURL != "" {
		result.BaseURL = child.BaseURL
	}

	if child.Timeout > 0 {
		result.Timeout = child.Timeout
	}
	if child.Retries > 0 {
		result.Retries = child.Retries
	}
	if child.Concurrency > 0 {
		result.Concurrency = child.Concurrency
	}

	result.Setup = append(parent.Setup, child.Setup...)

	result.Teardown = append(parent.Teardown, child.Teardown...)

	result.TestCases = append(parent.TestCases, child.TestCases...)

	result.Tags = mergeUniqueTags(parent.Tags, child.Tags)

	result.MockRules = append(parent.MockRules, child.MockRules...)

	if child.DataDriven != nil {
		result.DataDriven = child.DataDriven
	}

	return result
}

func deepMergeMaps(a, b map[string]any) map[string]any {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	result := make(map[string]any)

	for k, v := range a {
		result[k] = v
	}

	for k, vb := range b {
		if va, ok := result[k]; ok {
			mapA, okA := va.(map[string]any)
			mapB, okB := vb.(map[string]any)
			if okA && okB {
				result[k] = deepMergeMaps(mapA, mapB)
			} else {
				result[k] = vb
			}
		} else {
			result[k] = vb
		}
	}

	return result
}

func mergeUniqueTags(a, b []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(a)+len(b))

	for _, tag := range a {
		if !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}

	for _, tag := range b {
		if !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}

	return result
}
