package mcp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// confirmDecisionRequestID is the ID a confirmation is filed under, and
// confirmDecisionField the form field it asks for. The client echoes both back,
// so the two halves of one tool call must agree on them.
const (
	confirmDecisionRequestID = "confirm"
	confirmDecisionField     = "confirm"
)

// confirmationStatePurpose names this use of the server secret, so the key
// that signs a confirmation's request state signs nothing else.
const confirmationStatePurpose = "bb.mcp.confirmation.request_state"

// confirmableWrite is a write a tool asks the user to confirm before making it.
// The request state is signed for its procedure and resource, so an answer to
// one confirmation cannot be replayed for another.
type confirmableWrite interface {
	confirmedProcedure() string
	confirmedResource() string
}

// askToConfirm returns the result that asks the user to confirm w. The SDK
// elicits within this tools/call and re-invokes the handler with the answer
// and the request state, the same multi round-trip shape as
// elicitDatabaseChoice. A client on the 2026-07-28 protocol answers and echoes
// the state itself.
func (s *Server) askToConfirm(ctx context.Context, w confirmableWrite, digest []byte, message, title, description string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		RequestState: s.signConfirmationState(ctx, w, digest),
		InputRequests: mcp.InputRequestMap{
			confirmDecisionRequestID: &mcp.ElicitParams{
				Mode:    "form",
				Message: message,
				RequestedSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						confirmDecisionField: map[string]any{
							"type":        "boolean",
							"title":       title,
							"description": description,
						},
					},
					"required": []string{confirmDecisionField},
				},
			},
		},
	}
}

// confirmationAnswer returns the answer and request state a call carries, if
// it answers a confirmation.
func confirmationAnswer(req *mcp.CallToolRequest) (mcp.InputResponse, string, bool) {
	if req.Params == nil {
		return nil, "", false
	}
	response, answered := req.Params.InputResponses[confirmDecisionRequestID]
	return response, req.Params.RequestState, answered
}

// confirmationOutcome reports whether the answer confirms the write, and if
// not, why, in words the result can carry.
func confirmationOutcome(response mcp.InputResponse) (bool, string) {
	elicited, ok := response.(*mcp.ElicitResult)
	if !ok || elicited == nil {
		return false, "the confirmation answer could not be read"
	}
	switch elicited.Action {
	case "accept":
		if confirmed, ok := elicited.Content[confirmDecisionField].(bool); ok && confirmed {
			return true, ""
		}
		return false, "the form was submitted without the confirmation checked"
	case "decline":
		return false, "the user declined"
	case "cancel":
		return false, "the user cancelled"
	default:
		return false, fmt.Sprintf("the confirmation answer %q is not one Bytebase acts on", elicited.Action)
	}
}

// signConfirmationState returns the request state the form is sent with: a
// digest of what the user is asked about, signed together with who asks,
// through which client, and which write on which resource.
func (s *Server) signConfirmationState(ctx context.Context, w confirmableWrite, digest []byte) string {
	return base64.RawURLEncoding.EncodeToString(digest) + "." +
		base64.RawURLEncoding.EncodeToString(s.confirmationMAC(ctx, w, digest))
}

// verifyConfirmationState returns the digest a request state carries, and
// whether this server signed it for this caller and this write.
func (s *Server) verifyConfirmationState(ctx context.Context, w confirmableWrite, state string) ([]byte, bool) {
	encodedDigest, encodedMAC, found := strings.Cut(state, ".")
	if !found {
		return nil, false
	}
	digest, err := base64.RawURLEncoding.DecodeString(encodedDigest)
	if err != nil {
		return nil, false
	}
	mac, err := base64.RawURLEncoding.DecodeString(encodedMAC)
	if err != nil {
		return nil, false
	}
	return digest, hmac.Equal(mac, s.confirmationMAC(ctx, w, digest))
}

func (s *Server) confirmationMAC(ctx context.Context, w confirmableWrite, digest []byte) []byte {
	key := hmac.New(sha256.New, []byte(s.secret))
	_, _ = key.Write([]byte(confirmationStatePurpose))
	signed := strings.Join([]string{getWorkspaceID(ctx), getUserEmail(ctx), getOAuth2ClientID(ctx), w.confirmedProcedure(), w.confirmedResource(), ""}, "\x00")
	mac := hmac.New(sha256.New, key.Sum(nil))
	_, _ = mac.Write(append([]byte(signed), digest...))
	return mac.Sum(nil)
}
