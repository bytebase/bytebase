package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"time"

	"github.com/golang-jwt/jwt/v5"
	errs "github.com/pkg/errors"
)

const (
	// internalMCPAudience is the audience of the delegated credential minted at
	// the /mcp boundary for the private in-memory transport, and the only claim
	// that marks it as internal. No public listener ever accepts it. A separate
	// token_use would say nothing the audience does not: unlike the external MCP
	// token, whose audience is a per-deployment resource URI that no verifier
	// can match against a constant, this audience IS a constant.
	internalMCPAudience = "bb.mcp.internal"
	// internalMCPTokenDuration is the delegated credential's lifetime. It is
	// request-scale: minted per inbound MCP request, it only needs to cover one
	// tool execution (worst case a multi-call change tool with plan-check
	// polling), never a session.
	internalMCPTokenDuration = 5 * time.Minute
)

// DelegatedMCPCredential is the claim set of the credential minted at the /mcp
// boundary and carried on the private in-memory transport. Identity + grant
// state only: Scope and Resource are the grant's STORED values copied verbatim
// (empty for legacy sessions), and no roles are baked in — downstream
// authorization re-resolves membership and RBAC live, exactly as for a public
// request. This claim shape is the contract PR 5 parses into common.AuthContext.
type DelegatedMCPCredential struct {
	Principal     string
	WorkspaceID   string
	ClientID      string
	CorrelationID string
	Scope         string
	Resource      string
}

// internalMCPClaimsMessage is the JWT claim layout of the delegated credential.
type internalMCPClaimsMessage struct {
	claimsMessage
	ClientID      string `json:"client_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	Scope         string `json:"scope,omitempty"`
	GrantResource string `json:"grant_resource,omitempty"`
}

// internalMCPSigningKey derives the internal credential's signing key from the
// server secret.
//
// This is one of two independent layers separating internal from public
// credentials; the audience check is the other, and either alone is sufficient
// (verified by mutation: signing these with the raw secret leaves every
// public-surface rejection test green, and it fails only the wire-shape test
// that pins this derivation). Keep both. The credential shares the public kid,
// so a public verifier will try it, hand back the raw secret, and fail on the
// signature before it ever reads a claim.
//
// The derivation also means a leaked internal credential can never be
// re-signed into a public one.
func internalMCPSigningKey(secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(internalMCPAudience + ":" + keyID))
	return mac.Sum(nil)
}

// GenerateInternalMCPToken mints the delegated credential for one MCP request.
func GenerateInternalMCPToken(cred DelegatedMCPCredential, secret string) (string, error) {
	return generateInternalMCPTokenWithExpiry(cred, secret, time.Now().Add(internalMCPTokenDuration))
}

func generateInternalMCPTokenWithExpiry(cred DelegatedMCPCredential, secret string, expirationTime time.Time) (string, error) {
	claims := &internalMCPClaimsMessage{
		claimsMessage: claimsMessage{
			RegisteredClaims: jwt.RegisteredClaims{
				Audience:  jwt.ClaimStrings{internalMCPAudience},
				ExpiresAt: jwt.NewNumericDate(expirationTime),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    issuer,
				Subject:   cred.Principal,
			},
			WorkspaceID: cred.WorkspaceID,
		},
		ClientID:      cred.ClientID,
		CorrelationID: cred.CorrelationID,
		Scope:         cred.Scope,
		GrantResource: cred.Resource,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = keyID
	return token.SignedString(internalMCPSigningKey(secret))
}

// IsMCPOriginatedToken reports whether a bearer identifies MCP-originated
// traffic — either an external MCP OAuth2 token or the delegated credential
// minted at the /mcp boundary for the internal transport.
//
// Guards that must not serve MCP sessions have to ask this rather than
// inspecting claims themselves: since P1a PR 3 the external token's audience is
// a per-deployment resource URI (so token_use is the recognizer), and since PR
// 4 tool traffic presents the delegated credential instead, which is signed
// with a derived key and therefore invisible to ExtractClaimsFromExpiredToken.
func IsMCPOriginatedToken(tokenStr, secret string) bool {
	if tokenStr == "" {
		return false
	}
	if _, err := VerifyInternalMCPToken(tokenStr, secret); err == nil {
		return true
	}
	claims, err := ExtractClaimsFromExpiredToken(tokenStr, secret)
	if err != nil {
		return false
	}
	return claims.TokenUse == TokenUseMCP || audienceContains(claims.Audience, OAuth2AccessTokenAudience)
}

// VerifyInternalMCPToken validates a delegated credential and returns its
// claims. It accepts ONLY the internal credential: the derived key rejects
// everything minted for a public surface, and the audience is checked on top.
func VerifyInternalMCPToken(tokenStr, secret string) (*DelegatedMCPCredential, error) {
	claims := &internalMCPClaimsMessage{}
	if _, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, errs.Errorf("unexpected internal credential signing method=%v", t.Header["alg"])
		}
		if kid, ok := t.Header["kid"].(string); ok && kid == keyID {
			return internalMCPSigningKey(secret), nil
		}
		return nil, errs.Errorf("unexpected internal credential kid=%v", t.Header["kid"])
	}); err != nil {
		return nil, errs.Wrap(err, "invalid internal MCP credential")
	}
	if !audienceContains(claims.Audience, internalMCPAudience) {
		return nil, errs.Errorf("internal MCP credential audience mismatch, got %q", claims.Audience)
	}
	return &DelegatedMCPCredential{
		Principal:     claims.Subject,
		WorkspaceID:   claims.WorkspaceID,
		ClientID:      claims.ClientID,
		CorrelationID: claims.CorrelationID,
		Scope:         claims.Scope,
		Resource:      claims.GrantResource,
	}, nil
}
