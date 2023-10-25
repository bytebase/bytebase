// Package secret includes the component of getting secrets from external sources.
package secret

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

var secretCache = make(map[string]string)

// ReplaceExternalSecret replaces the secret with external secret.
func ReplaceExternalSecret(secret string) (string, error) {
	ok, secretURL := GetExternalSecretURL(secret)
	if !ok {
		return secret, nil
	}
	if v, ok := secretCache[secretURL]; ok {
		return v, nil
	}
	secret, err := getSecretFromURL(secretURL)
	if err != nil {
		return "", err
	}
	secretCache[secretURL] = secret
	return secret, err
}

// GetExternalSecretURL gets external secret URL from secret.
func GetExternalSecretURL(secret string) (bool, string) {
	if !strings.HasPrefix(secret, "{{") {
		return false, ""
	}
	if !strings.HasSuffix(secret, "}}") {
		return false, ""
	}
	s := secret[2 : len(secret)-2]
	if _, err := url.ParseRequestURI(s); err != nil {
		return false, ""
	}
	return true, s
}

type payload struct {
	Data string `json:"data"`
}

type accessResponse struct {
	Payload payload `json:"payload"`
}

func getSecretFromURL(secretURL string) (string, error) {
	response, err := http.Get(secretURL)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get secret from %q", secretURL)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", errors.Wrapf(err, "failed to get secret from %q status %v", secretURL, response.StatusCode)
	}

	var r accessResponse
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&r); err != nil {
		return "", errors.Wrapf(err, "failed to decode JSON response")
	}
	secret, err := base64.StdEncoding.DecodeString(r.Payload.Data)
	if err != nil {
		return "", errors.Wrapf(err, "failed to base64 decode secret")
	}
	return string(secret), nil
}
