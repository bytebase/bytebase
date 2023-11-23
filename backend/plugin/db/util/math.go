package util

import "github.com/pkg/errors"

// CeilDivision returns the smallest integer value greater than or equal to the
// result of dividing the first argument by the second argument.
// For example, CeilDivision(5, 2) returns 3.
func CeilDivision(dividend int, divisor int) (int, error) {
	if divisor == 0 {
		return 0, errors.New("divisor cannot be 0")
	}
	if dividend == 0 {
		return 0, nil
	}
	return (dividend + divisor - 1) / divisor, nil
}
