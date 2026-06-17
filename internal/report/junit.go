package report

import (
	"encoding/xml"
	"fmt"

	"apitester/internal/models"
)

type JUnitReporter struct{}

func NewJUnitReporter() *JUnitReporter {
	return &JUnitReporter{}
}

type junitTestsuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestsuite `xml:"testsuite"`
}

type junitTestsuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Passed    int             `xml:"passed,attr,omitempty"`
	Failures  int             `xml:"failures,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      float64         `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr,omitempty"`
	Testcases []junitTestcase `xml:"testcase"`
}

type junitTestcase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
}

type junitFailure struct {
	XMLName xml.Name `xml:"failure"`
	Message string   `xml:"message,attr"`
	Type    string   `xml:"type,attr"`
	Content string   `xml:",cdata"`
}

type junitSkipped struct {
	XMLName xml.Name `xml:"skipped"`
	Message string   `xml:"message,attr,omitempty"`
}

func (r *JUnitReporter) Generate(suite models.SuiteResult) (string, error) {
	testcases := make([]junitTestcase, 0, len(suite.Tests))

	for _, test := range suite.Tests {
		tc := junitTestcase{
			Name:      test.Name,
			Classname: suite.Name,
			Time:      test.Duration.Seconds(),
		}

		if test.Skipped {
			tc.Skipped = &junitSkipped{
				Message: test.SkipReason,
			}
		} else if !test.Passed {
			failureMsg := r.buildFailureMessage(test)
			tc.Failure = &junitFailure{
				Message: failureMsg,
				Type:    "AssertionError",
				Content: r.buildFailureContent(test),
			}
		}

		testcases = append(testcases, tc)
	}

	testsuite := junitTestsuite{
		Name:      suite.Name,
		Tests:     suite.Total,
		Passed:    suite.Passed,
		Failures:  suite.Failed,
		Skipped:   suite.Skipped,
		Time:      suite.Duration.Seconds(),
		Timestamp: suite.StartTime.Format("2006-01-02T15:04:05"),
		Testcases: testcases,
	}

	testsuites := junitTestsuites{
		Suites: []junitTestsuite{testsuite},
	}

	data, err := xml.MarshalIndent(testsuites, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal xml: %w", err)
	}

	return xml.Header + string(data), nil
}

func (r *JUnitReporter) buildFailureMessage(test models.TestResult) string {
	if test.Error != "" {
		return test.Error
	}

	for _, assert := range test.Assertions {
		if !assert.Passed {
			return fmt.Sprintf("断言失败: %s", assert.Name)
		}
	}

	return "测试失败"
}

func (r *JUnitReporter) buildFailureContent(test models.TestResult) string {
	var content string

	if test.Error != "" {
		content += fmt.Sprintf("错误信息:\n%s\n\n", test.Error)
	}

	if len(test.Assertions) > 0 {
		content += "断言详情:\n"
		for i, assert := range test.Assertions {
			status := "✓"
			if !assert.Passed {
				status = "✗"
			}
			content += fmt.Sprintf("%d. [%s] %s\n", i+1, status, assert.Name)
			if !assert.Passed {
				content += fmt.Sprintf("   期望值: %v\n", assert.Expected)
				content += fmt.Sprintf("   实际值: %v\n", assert.Actual)
				if assert.Description != "" {
					content += fmt.Sprintf("   描述: %s\n", assert.Description)
				}
			}
		}
	}

	if test.Request != nil {
		content += fmt.Sprintf("\n请求: %s %s\n", test.Request.Method, test.Request.URL)
	}

	if test.Response != nil {
		content += fmt.Sprintf("响应状态码: %d\n", test.Response.StatusCode)
	}

	return content
}
