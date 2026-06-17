package filter

import (
	"apitester/internal/models"
)

type FilterEngine struct {
	exprParser *ExpressionParser
}

func NewFilterEngine() *FilterEngine {
	return &FilterEngine{
		exprParser: NewExpressionParser(),
	}
}

func (fe *FilterEngine) FilterCases(cases []*models.TestCase, options *models.ExecutionOptions) []*models.TestCase {
	if options == nil {
		return cases
	}

	var filtered []*models.TestCase

	for _, tc := range cases {
		if tc.Skip {
			continue
		}

		if !MatchTags(tc.Tags, options.Tags, options.SkipTags) {
			continue
		}

		if options.Filter != "" {
			if !fe.matchCaseFilter(tc, options.Filter) {
				continue
			}
		}

		filtered = append(filtered, tc)
	}

	return filtered
}

func (fe *FilterEngine) FilterResults(results []*models.TestResult, expression string) ([]*models.TestResult, error) {
	if expression == "" {
		return results, nil
	}

	expr, err := fe.exprParser.ParseExpression(expression)
	if err != nil {
		return nil, err
	}

	var filtered []*models.TestResult
	for _, r := range results {
		if fe.exprParser.Evaluate(expr, r) {
			filtered = append(filtered, r)
		}
	}

	return filtered, nil
}

func (fe *FilterEngine) matchCaseFilter(tc *models.TestCase, filter string) bool {
	expr, err := fe.exprParser.ParseExpression(filter)
	if err != nil {
		return false
	}

	simulatedResult := &models.TestResult{
		CaseID:   tc.ID,
		CaseName: tc.Name,
	}

	if fe.exprParser.Evaluate(expr, simulatedResult) {
		return true
	}

	for _, tag := range tc.Tags {
		tagResult := &models.TestResult{
			CaseID:   tc.ID,
			CaseName: tag,
		}
		if fe.exprParser.Evaluate(expr, tagResult) {
			return true
		}
	}

	return false
}
