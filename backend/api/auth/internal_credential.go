package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"time"

	"github.com/golang-jwt/jwt/v5"
	errs "github.com/pkg/errors"
)

const (
	// InternalMCPAudience is the audience of the delegated credential minted at
	// the /mcp boundary for the private in-memory transport. No public listener
	// ever accepts it.
	InternalMCPAudience = "bb.mcp.internal"
	// TokenUseMCPInternal marks the delegated credential's token_use claim,
	// distinct from TokenUseMCP so the general API's token_use recognition
	// never admits it.
	TokenUseMCPInternal = "mcp_internal"
	// internalMCPKeyID is the kid of the delegated credential. The public
	// keyfuncs only ever return a key for kid "v1", so an internal credential
	// presented on a public surface fails before signature verification.
	internalMCPKeyID = "mcp-internal-v1"
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
// server secret. Deriving (instead of reusing the secret raw) means that even a
// hypothetical kid-check bypass on a public surface still fails signature
// verification, and a leaked internal credential can never be re-signed into a
// public one.
func internalMCPSigningKey(secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(InternalMCPAudience + ":" + internalMCPKeyID))
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
				Audience:  jwt.ClaimStrings{InternalMCPAudience},
				ExpiresAt: jwt.NewNumericDate(expirationTime),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    issuer,
				Subject:   cred.Principal,
			},
			WorkspaceID: cred.WorkspaceID,
			TokenUse:    TokenUseMCPInternal,
		},
		ClientID:      cred.ClientID,
		CorrelationID: cred.CorrelationID,
		Scope:         cred.Scope,
		GrantResource: cred.Resource,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = internalMCPKeyID
	return token.SignedString(internalMCPSigningKey(secret))
}

// VerifyInternalMCPToken validates a delegated credential and returns its
// claims. It accepts ONLY the internal credential: the dedicated kid selects
// the derived key (so tokens signed with the raw secret fail), and audience and
// token_use are checked strictly on top.
func VerifyInternalMCPToken(tokenStr, secret string) (*DelegatedMCPCredential, error) {
	claims := &internalMCPClaimsMessage{}
	if _, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, errs.Errorf("unexpected internal credential signing method=%v", t.Header["alg"])
		}
		if kid, ok := t.Header["kid"].(string); ok && kid == internalMCPKeyID {
			return internalMCPSigningKey(secret), nil
		}
		return nil, errs.Errorf("unexpected internal credential kid=%v", t.Header["kid"])
	}); err != nil {
		return nil, errs.Wrap(err, "invalid internal MCP credential")
	}
	if !audienceContains(claims.Audience, InternalMCPAudience) {
		return nil, errs.Errorf("internal MCP credential audience mismatch, got %q", claims.Audience)
	}
	if claims.TokenUse != TokenUseMCPInternal {
		return nil, errs.Errorf("internal MCP credential token_use mismatch, got %q", claims.TokenUse)
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
