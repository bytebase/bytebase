package mcp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

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

// confirmationLifetime bounds how long after asking an answer is acted on. The
// signed state carries no nonce, and a write's digest repeats whenever the same
// write is asked again, so without it an accepted answer could be replayed
// once the write it confirmed became possible again.
const confirmationLifetime = 10 * time.Minute

// confirmableWrite is a write a tool asks the user to confirm before making it.
// The request state is signed for its procedure and resource, so an answer to
// one confirmation cannot be replayed for another.
type confirmableWrite interface {
	confirmedProcedure() string
	confirmedResource() string
}

// canShowConfirmation reports whether the client declared the form a
// confirmation needs. A tool checks it before reading any answer: a client that
// declared no form cannot have shown one, so an answer it sends is not a
// confirmation.
func canShowConfirmation(req *mcp.CallToolRequest) bool {
	return req != nil && req.Session != nil && clientSupportsElicitation(req.Session)
}

// askToConfirm returns the result that asks the user to confirm w. The SDK
// elicits within this tools/call and re-invokes the handler with the answer
// and the request state, the same multi round-trip shape as
// elicitDatabaseChoice. A client on the 2026-07-28 protocol answers and echoes
// the state itself.
func (s *Server) askToConfirm(ctx context.Context, w confirmableWrite, digest []byte, message, title, description string) *mcp.CallToolResult {
	return s.askWithForm(ctx, w, digest, message, map[string]any{
		"type": "object",
		"properties": map[string]any{
			confirmDecisionField: map[string]any{
				"type":        "boolean",
				"title":       title,
				"description": description,
			},
		},
		"required": []string{confirmDecisionField},
	})
}

// askWithForm is askToConfirm with a form the caller builds.
func (s *Server) askWithForm(ctx context.Context, w confirmableWrite, digest []byte, message string, schema map[string]any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		RequestState: s.signConfirmationState(ctx, w, digest, time.Now()),
		InputRequests: mcp.InputRequestMap{
			confirmDecisionRequestID: &mcp.ElicitParams{
				Mode:            "form",
				Message:         message,
				RequestedSchema: schema,
			},
		},
	}
}

// confirmationAnswer returns the answer and request state a call carries, if
// it answers a confirmation.
func confirmationAnswer(req *mcp.CallToolRequest) (mcp.InputResponse, string, bool) {
	if req == nil || req.Params == nil {
		return nil, "", false
	}
	response, answered := req.Params.InputResponses[confirmDecisionRequestID]
	return response, req.Params.RequestState, answered
}

// confirmationReply is a call's answer to a confirmation this server sent.
type confirmationReply struct {
	response mcp.InputResponse
	// matches is whether the user was asked about the write the call would make.
	matches bool
	// expired is whether the user was asked longer ago than confirmationLifetime.
	expired bool
}

// signedReply returns the call's answer to a confirmation this server signed
// for this caller and write, and whether it carries one. An answer without such
// a state is no answer: a client can send one on a call that was never asked,
// so the call is asked like a first one.
func (s *Server) signedReply(ctx context.Context, req *mcp.CallToolRequest, w confirmableWrite, digest []byte) (*confirmationReply, bool) {
	response, state, answered := confirmationAnswer(req)
	if !answered {
		return nil, false
	}
	asked, askedAt, signed := s.verifyConfirmationState(ctx, w, state)
	if !signed {
		return nil, false
	}
	return &confirmationReply{
		response: response,
		matches:  bytes.Equal(asked, digest),
		expired:  time.Since(askedAt) > confirmationLifetime,
	}, true
}

// staleReason says why a confirmed answer cannot be acted on, or "" when it
// can. mismatch is the tool's words for an answer about a different write. An
// expired answer is reported, not asked again: an older client gets one second
// round per call, so a form sent there would go unanswered.
func (r *confirmationReply) staleReason(mismatch string) string {
	switch {
	case r.expired:
		return fmt.Sprintf("the user was asked more than %d minutes ago", int(confirmationLifetime.Minutes()))
	case !r.matches:
		return mismatch
	default:
		return ""
	}
}

// confirmationOutcome reports whether the answer confirms the write, and if
// not, why, in words the result can carry.
func confirmationOutcome(response mcp.InputResponse) (bool, string) {
	content, accepted, why := acceptedForm(response)
	if !accepted {
		return false, why
	}
	if confirmed, ok := content[confirmDecisionField].(bool); ok && confirmed {
		return true, ""
	}
	return false, "the form was submitted without the confirmation checked"
}

// acceptedForm returns what an accepted form holds, or why the answer accepts
// nothing.
func acceptedForm(response mcp.InputResponse) (map[string]any, bool, string) {
	elicited, ok := response.(*mcp.ElicitResult)
	if !ok || elicited == nil {
		return nil, false, "the confirmation answer could not be read"
	}
	switch elicited.Action {
	case "accept":
		return elicited.Content, true, ""
	case "decline":
		return nil, false, "the user declined"
	case "cancel":
		return nil, false, "the user cancelled"
	default:
		return nil, false, fmt.Sprintf("the confirmation answer %q is not one Bytebase acts on", elicited.Action)
	}
}

// signConfirmationState returns the request state the form is sent with: a
// digest of what the user is asked about and when they were asked, signed
// together with who asks, through which client, and which write on which
// resource.
func (s *Server) signConfirmationState(ctx context.Context, w confirmableWrite, digest []byte, askedAt time.Time) string {
	askedAtUnix := strconv.FormatInt(askedAt.Unix(), 10)
	return base64.RawURLEncoding.EncodeToString(digest) + "." + askedAtUnix + "." +
		base64.RawURLEncoding.EncodeToString(s.confirmationMAC(ctx, w, digest, askedAtUnix))
}

// verifyConfirmationState returns the digest a request state carries, when the
// user was asked, and whether this server signed it for this caller and this
// write.
func (s *Server) verifyConfirmationState(ctx context.Context, w confirmableWrite, state string) ([]byte, time.Time, bool) {
	parts := strings.Split(state, ".")
	if len(parts) != 3 {
		return nil, time.Time{}, false
	}
	digest, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, time.Time{}, false
	}
	askedAtUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, time.Time{}, false
	}
	mac, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, time.Time{}, false
	}
	if !hmac.Equal(mac, s.confirmationMAC(ctx, w, digest, parts[1])) {
		return nil, time.Time{}, false
	}
	return digest, time.Unix(askedAtUnix, 0), true
}

func (s *Server) confirmationMAC(ctx context.Context, w confirmableWrite, digest []byte, askedAtUnix string) []byte {
	key := hmac.New(sha256.New, []byte(s.secret))
	_, _ = key.Write([]byte(confirmationStatePurpose))
	signed := strings.Join([]string{getWorkspaceID(ctx), getUserEmail(ctx), getOAuth2ClientID(ctx), w.confirmedProcedure(), w.confirmedResource(), askedAtUnix, ""}, "\x00")
	mac := hmac.New(sha256.New, key.Sum(nil))
	_, _ = mac.Write(append([]byte(signed), digest...))
	return mac.Sum(nil)
}
