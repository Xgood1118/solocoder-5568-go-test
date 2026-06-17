package assert

import (
	"apitester/internal/assert/assertions"
)

func Compare(operator string, actual, expected any) (bool, error) {
	return assertions.Compare(operator, actual, expected)
}
