package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyPKCE(t *testing.T) {
	a := require.New(t)

	// Generate a valid code_verifier and code_challenge
	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	tests := []struct {
		name                string
		codeVerifier        string
		codeChallenge       string
		codeChallengeMethod string
		wantErr             bool
		errContains         string
	}{
		{
			name:                "valid S256",
			codeVerifier:        codeVerifier,
			codeChallenge:       codeChallenge,
			codeChallengeMethod: "S256",
			wantErr:             false,
		},
		{
			name:                "invalid verifier",
			codeVerifier:        "wrongverifier",
			codeChallenge:       codeChallenge,
			codeChallengeMethod: "S256",
			wantErr:             true,
			errContains:         "does not match",
		},
		{
			name:                "unsupported method plain",
			codeVerifier:        codeVerifier,
			codeChallenge:       codeChallenge,
			codeChallengeMethod: "plain",
			wantErr:             true,
			errContains:         "only S256",
		},
		{
			name:                "empty method",
			codeVerifier:        codeVerifier,
			codeChallenge:       codeChallenge,
			codeChallengeMethod: "",
			wantErr:             true,
			errContains:         "only S256",
		},
		{
			name:                "empty challenge",
			codeVerifier:        codeVerifier,
			codeChallenge:       "",
			codeChallengeMethod: "S256",
			wantErr:             true,
			errContains:         "does not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			err := VerifyPKCE(tt.codeVerifier, tt.codeChallenge, tt.codeChallengeMethod)
			if tt.wantErr {
				a.Error(err)
				if tt.errContains != "" {
					a.Contains(err.Error(), tt.errContains)
				}
			} else {
				a.NoError(err)
			}
		})
	}
}

func TestValidateCodeVerifier(t *testing.T) {
	a := require.New(t)

	tests := []struct {
		name        string
		verifier    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid 43 characters",
			verifier: "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			wantErr:  false,
		},
		{
			name:     "valid 128 characters",
			verifier: strings.Repeat("a", 128),
			wantErr:  false,
		},
		{
			name:     "valid with all unreserved chars",
			verifier: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~",
			wantErr:  false,
		},
		{
			name:        "too short 42 characters",
			verifier:    strings.Repeat("a", 42),
			wantErr:     true,
			errContains: "43-128 characters",
		},
		{
			name:        "too long 129 characters",
			verifier:    strings.Repeat("a", 129),
			wantErr:     true,
			errContains: "43-128 characters",
		},
		{
			name:        "invalid characters space",
			verifier:    "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFO EjXk",
			wantErr:     true,
			errContains: "invalid characters",
		},
		{
			name:        "invalid characters plus",
			verifier:    "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFO+EjXk",
			wantErr:     true,
			errContains: "invalid characters",
		},
		{
			name:        "invalid characters equals",
			verifier:    "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFO=EjXk",
			wantErr:     true,
			errContains: "invalid characters",
		},
		{
			name:        "empty string",
			verifier:    "",
			wantErr:     true,
			errContains: "43-128 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			err := ValidateCodeVerifier(tt.verifier)
			if tt.wantErr {
				a.Error(err)
				if tt.errContains != "" {
					a.Contains(err.Error(), tt.errContains)
				}
			} else {
				a.NoError(err)
			}
		})
	}
}

func TestValidateCodeChallenge(t *testing.T) {
	a := require.New(t)

	// Generate a valid 43-character base64url challenge
	validChallenge := base64.RawURLEncoding.EncodeToString(make([]byte, 32))

	tests := []struct {
		name        string
		challenge   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid 43 characters",
			challenge: validChallenge,
			wantErr:   false,
		},
		{
			name:        "too short 42 characters",
			challenge:   strings.Repeat("a", 42),
			wantErr:     true,
			errContains: "43 characters",
		},
		{
			name:        "too long 44 characters",
			challenge:   strings.Repeat("a", 44),
			wantErr:     true,
			errContains: "43 characters",
		},
		{
			name:        "contains plus sign",
			challenge:   "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuG+EpNc3cm",
			wantErr:     true,
			errContains: "base64url encoded",
		},
		{
			name:        "contains slash",
			challenge:   "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuG/EpNc3cm",
			wantErr:     true,
			errContains: "base64url encoded",
		},
		{
			name:        "contains padding",
			challenge:   "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuG=EpNc3cm",
			wantErr:     true,
			errContains: "base64url encoded",
		},
		{
			name:        "empty string",
			challenge:   "",
			wantErr:     true,
			errContains: "43 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			err := ValidateCodeChallenge(tt.challenge)
			if tt.wantErr {
				a.Error(err)
				if tt.errContains != "" {
					a.Contains(err.Error(), tt.errContains)
				}
			} else {
				a.NoError(err)
			}
		})
	}
}

func TestIsUnreserved(t *testing.T) {
	a := require.New(t)

	// Test valid unreserved characters
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	for _, c := range validChars {
		a.True(isUnreserved(c), "character %c should be unreserved", c)
	}

	// Test invalid characters
	invalidChars := " !@#$%^&*()+=[]{}|\\:;\"'<>,?/`"
	for _, c := range invalidChars {
		a.False(isUnreserved(c), "character %c should not be unreserved", c)
	}
}
