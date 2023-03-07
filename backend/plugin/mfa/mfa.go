package mfa

import (
	"crypto/rand"
	"math/big"

	"github.com/pkg/errors"
)

// randomString generates a random string of length n.
func randomString(n int) (string, error) {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	randomInt := func(max *big.Int) (int, error) {
		r, err := rand.Int(rand.Reader, max)
		if err != nil {
			return 0, err
		}
		return int(r.Int64()), nil
	}

	buffer := make([]byte, n)
	max := big.NewInt(int64(len(alphanum)))
	for i := 0; i < n; i++ {
		index, err := randomInt(max)
		if err != nil {
			return "", err
		}
		buffer[i] = alphanum[index]
	}

	return string(buffer), nil
}

// GenerateRecoveryCodes generates n recovery codes.
func GenerateRecoveryCodes(n int) ([]string, error) {
	recoveryCodes := make([]string, n)
	for i := 0; i < n; i++ {
		code, err := randomString(10)
		if err != nil {
			return nil, errors.Wrap(err, "generate random characters")
		}
		recoveryCodes[i] = code
	}
	return recoveryCodes, nil
}
