package store

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// AccountMessage is a unified account representation for the auth layer.
// It contains only the fields needed for authentication and authorization,
// regardless of the underlying principal type (END_USER, SERVICE_ACCOUNT, WORKLOAD_IDENTITY).
type AccountMessage struct {
	Email string
	Name  string
	Type  storepb.PrincipalType
	// Workspace is the workspace resource ID.
	// Empty for END_USER (global identity, workspace resolved via IAM policy).
	// Populated for SERVICE_ACCOUNT and WORKLOAD_IDENTITY (workspace-scoped).
	Workspace     string
	PasswordHash  string
	MemberDeleted bool
}

// GetAccountByEmail gets any principal (user, service account, or workload identity) by email.
// Used by the auth layer (login, token validation) and runners.
// Queries cross-workspace — safe because all PKs (email, id) are globally unique.
func (s *Store) GetAccountByEmail(ctx context.Context, email string) (*AccountMessage, error) {
	if common.IsServiceAccountEmail(email) {
		sa, err := s.GetServiceAccountByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if sa == nil {
			return nil, nil
		}
		return &AccountMessage{
			Email:         sa.Email,
			Name:          sa.Name,
			Workspace:     sa.Workspace,
			PasswordHash:  sa.ServiceKeyHash,
			Type:          storepb.PrincipalType_SERVICE_ACCOUNT,
			MemberDeleted: sa.MemberDeleted,
		}, nil
	}

	if common.IsWorkloadIdentityEmail(email) {
		wi, err := s.GetWorkloadIdentityByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if wi == nil {
			return nil, nil
		}
		return &AccountMessage{
			Email:         wi.Email,
			Name:          wi.Name,
			Workspace:     wi.Workspace,
			Type:          storepb.PrincipalType_WORKLOAD_IDENTITY,
			MemberDeleted: wi.MemberDeleted,
		}, nil
	}

	// Default to end user lookup.
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}
	return &AccountMessage{
		Email:         user.Email,
		Name:          user.Name,
		Type:          storepb.PrincipalType_END_USER,
		PasswordHash:  user.PasswordHash,
		MemberDeleted: user.MemberDeleted,
	}, nil
}

// ResolvePrincipalAsUser converts an AccountMessage into a *UserMessage suitable
// for downstream code that expects the legacy single-table principal shape.
//
// For END_USER, it loads the full record from the principal table. For service
// accounts and workload identities (which live in separate tables and have no
// profile/MFA data), it returns a minimal UserMessage with Profile and MFAConfig
// zero-initialized so downstream code dereferencing those fields does not panic.
//
// Returns (nil, nil) if account.Type == END_USER and no matching user exists.
// Callers map this to their own not-found error.
func (s *Store) ResolvePrincipalAsUser(ctx context.Context, account *AccountMessage) (*UserMessage, error) {
	if account.Type == storepb.PrincipalType_END_USER {
		user, err := s.GetUserByEmail(ctx, account.Email)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get user %q", account.Email)
		}
		return user, nil
	}
	return &UserMessage{
		Email:         account.Email,
		Name:          account.Name,
		Type:          account.Type,
		MemberDeleted: account.MemberDeleted,
		Profile:       &storepb.UserProfile{},
		MFAConfig:     &storepb.MFAConfig{},
	}, nil
}
